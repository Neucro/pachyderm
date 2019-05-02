package collection

// Copyright 2016 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// We copy this code from etcd because the etcd implementation of STM does
// not have the DelAll method, which we need.

import (
	"fmt"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/pachyderm/pachyderm/src/client/pkg/tracing"
	"golang.org/x/net/context"
)

// STM is an interface for software transactional memory.
type STM interface {
	// Get returns the value for a key and inserts the key in the txn's read set.
	// If Get fails, it aborts the transaction with an error, never returning.
	Get(key string) (string, error)
	// Put adds a value for a key to the write set.
	Put(key, val string, ttl int64, ptr uintptr) error
	// Rev returns the revision of a key in the read set.
	Rev(key string) int64
	// Del deletes a key.
	Del(key string)
	// TTL returns the remaining time to live for 'key', or 0 if 'key' has no TTL
	TTL(key string) (int64, error)
	// DelAll deletes all keys with the given prefix
	// Note that the current implementation of DelAll is incomplete.
	// To use DelAll safely, do not issue any Get/Put operations after
	// DelAll is called.
	DelAll(key string)
	Context() context.Context
	// SetSafePutCheck sets the bit pattern to check if a put is safe.
	SetSafePutCheck(key string, ptr uintptr)
	// IsSafePut checks against the bit pattern for a key to see if it is safe to put.
	IsSafePut(key string, ptr uintptr) bool

	// commit attempts to apply the txn's changes to the server.
	commit() *v3.TxnResponse
	reset()
	fetch(key string) *v3.GetResponse
}

// stmError safely passes STM errors through panic to the STM error channel.
type stmError struct{ err error }

// NewSTM intiates a new STM operation. It uses a serializable model.
func NewSTM(ctx context.Context, c *v3.Client, apply func(STM) error) (*v3.TxnResponse, error) {
	return newSTMSerializable(ctx, c, apply)
}

// newSTMRepeatable initiates new repeatable read transaction; reads within
// the same transaction attempt always return the same data.
func newSTMRepeatable(ctx context.Context, c *v3.Client, apply func(STM) error) (*v3.TxnResponse, error) {
	s := &stm{client: c, ctx: ctx, getOpts: []v3.OpOption{v3.WithSerializable()}}
	return runSTM(s, apply)
}

// newSTMSerializable initiates a new serialized transaction; reads within the
// same transactiona attempt return data from the revision of the first read.
func newSTMSerializable(ctx context.Context, c *v3.Client, apply func(STM) error) (*v3.TxnResponse, error) {
	s := &stmSerializable{
		stm:      stm{client: c, ctx: ctx},
		prefetch: make(map[string]*v3.GetResponse),
	}
	return runSTM(s, apply)
}

// newSTMReadCommitted initiates a new read committed transaction.
func newSTMReadCommitted(ctx context.Context, c *v3.Client, apply func(STM) error) (*v3.TxnResponse, error) {
	s := &stmReadCommitted{stm{client: c, ctx: ctx, getOpts: []v3.OpOption{v3.WithSerializable()}}}
	return runSTM(s, apply)
}

type stmResponse struct {
	resp *v3.TxnResponse
	err  error
}

func runSTM(s STM, apply func(STM) error) (*v3.TxnResponse, error) {
	outc := make(chan stmResponse, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e, ok := r.(stmError)
				if !ok {
					// client apply panicked
					panic(r)
				}
				outc <- stmResponse{nil, e.err}
			}
		}()
		var out stmResponse
		for {
			s.reset()
			if out.err = apply(s); out.err != nil {
				break
			}
			if out.resp = s.commit(); out.resp != nil {
				break
			}
		}
		outc <- out
	}()
	r := <-outc
	return r.resp, r.err
}

// stm implements repeatable-read software transactional memory over etcd
type stm struct {
	client *v3.Client
	ctx    context.Context
	// rset holds read key values and revisions
	rset map[string]*v3.GetResponse
	// wset holds overwritten keys and their values
	wset map[string]stmPut
	// getOpts are the opts used for gets. Includes revision of first read for
	// stmSerializable
	getOpts []v3.OpOption
	// ttlset is a cache from key to lease TTL. It's similar to rset in that it
	// caches leases that have already been read, but each may contain keys not in
	// the other (ttlset in particular caches the TTL of all keys associated with
	// a lease after reading that lease, even if the other keys haven't been read)
	ttlset map[string]int64
	// newLeases is a map from TTL to lease ID; it caches new leases used for this
	// write. We de-dupe leases by TTL (values written with the same TTL get the
	// same lease) so that kvs in a collection and its indexes all share a lease.
	// It's similar to wset for TTLs.
	newLeases map[int64]v3.LeaseID
}

type stmPut struct {
	val        string
	ttl        int64
	op         v3.Op
	safePutPtr uintptr
}

func (s *stm) Context() context.Context {
	return s.ctx
}

func (s *stm) Get(key string) (string, error) {
	if wv, ok := s.wset[key]; ok {
		return wv.val, nil
	}
	return respToValue(key, s.fetch(key))
}

func (s *stm) SetSafePutCheck(key string, ptr uintptr) {
	if wv, ok := s.wset[key]; ok {
		wv.safePutPtr = ptr
		s.wset[key] = wv
	}
}

func (s *stm) IsSafePut(key string, ptr uintptr) bool {
	if _, ok := s.wset[key]; ok && s.wset[key].safePutPtr != 0 && ptr != s.wset[key].safePutPtr {
		return false
	}
	return true
}

func (s *stm) Put(key, val string, ttl int64, ptr uintptr) error {
	var options []v3.OpOption
	if ttl > 0 {
		lease, ok := s.newLeases[ttl]
		if !ok {
			span, ctx := tracing.AddSpanToAnyExisting(s.ctx, "/etcd/Grant")
			defer tracing.FinishAnySpan(span)
			leaseResp, err := s.client.Grant(ctx, ttl)
			if err != nil {
				return fmt.Errorf("error granting lease: %v", err)
			}
			lease = leaseResp.ID
			s.newLeases[ttl] = lease
			s.ttlset[key] = ttl // cache key->ttl, in case it's read later
		}
		options = append(options, v3.WithLease(lease))
	}
	s.wset[key] = stmPut{val, ttl, v3.OpPut(key, val, options...), ptr}
	return nil
}

func (s *stm) Del(key string) { s.wset[key] = stmPut{"", 0, v3.OpDelete(key), 0} }

func (s *stm) DelAll(key string) { s.wset[key] = stmPut{"", 0, v3.OpDelete(key, v3.WithPrefix()), 0} }

func (s *stm) Rev(key string) int64 {
	if resp := s.fetch(key); resp != nil && len(resp.Kvs) != 0 {
		return resp.Kvs[0].ModRevision
	}
	return 0
}

func (s *stm) commit() *v3.TxnResponse {
	span, ctx := tracing.AddSpanToAnyExisting(s.ctx, "/etcd/Txn")
	tracing.FinishAnySpan(span)

	cmps := s.cmps()
	puts := s.puts()
	txnresp, err := s.client.Txn(ctx).If(cmps...).Then(puts...).Commit()
	if err == rpctypes.ErrTooManyOps {
		panic(stmError{
			fmt.Errorf(
				"%v (%d comparisons, %d puts: hint: set --max-txn-ops on the "+
					"ETCD cluster to at least the largest of those values)",
				err, len(cmps), len(puts)),
		})
	} else if err != nil {
		panic(stmError{err})
	}
	if txnresp.Succeeded {
		return txnresp
	}
	return nil
}

// cmps guards the txn from updates to read set
func (s *stm) cmps() []v3.Cmp {
	cmps := make([]v3.Cmp, 0, len(s.rset))
	for k, rk := range s.rset {
		cmps = append(cmps, isKeyCurrent(k, rk))
	}
	return cmps
}

func (s *stm) fetch(key string) *v3.GetResponse {
	if resp, ok := s.rset[key]; ok {
		return resp
	}

	span, ctx := tracing.AddSpanToAnyExisting(s.ctx, "/etcd/Get", "key", key)
	defer tracing.FinishAnySpan(span)
	resp, err := s.client.Get(ctx, key, s.getOpts...)
	if err != nil {
		panic(stmError{err})
	}
	s.rset[key] = resp
	return resp
}

// puts is the list of ops for all pending writes
func (s *stm) puts() []v3.Op {
	puts := make([]v3.Op, 0, len(s.wset))
	for _, v := range s.wset {
		puts = append(puts, v.op)
	}
	return puts
}

func (s *stm) reset() {
	s.rset = make(map[string]*v3.GetResponse)
	s.wset = make(map[string]stmPut)
	s.ttlset = make(map[string]int64)
	s.newLeases = make(map[int64]v3.LeaseID)
}

type stmSerializable struct {
	stm
	prefetch map[string]*v3.GetResponse
}

func (s *stmSerializable) Get(key string) (string, error) {
	if wv, ok := s.wset[key]; ok {
		return wv.val, nil
	}
	return respToValue(key, s.fetch(key))
}

func (s *stmSerializable) fetch(key string) *v3.GetResponse {
	firstRead := len(s.rset) == 0
	if resp, ok := s.prefetch[key]; ok {
		delete(s.prefetch, key)
		s.rset[key] = resp
	}
	resp := s.stm.fetch(key)
	if firstRead {
		// txn's base revision is defined by the first read
		s.getOpts = []v3.OpOption{
			v3.WithRev(resp.Header.Revision),
			v3.WithSerializable(),
		}
	}
	return resp
}

func (s *stmSerializable) Rev(key string) int64 {
	s.Get(key)
	return s.stm.Rev(key)
}

func (s *stmSerializable) gets() ([]string, []v3.Op) {
	keys := make([]string, 0, len(s.rset))
	ops := make([]v3.Op, 0, len(s.rset))
	for k := range s.rset {
		keys = append(keys, k)
		ops = append(ops, v3.OpGet(k))
	}
	return keys, ops
}

func (s *stmSerializable) commit() *v3.TxnResponse {
	span, ctx := tracing.AddSpanToAnyExisting(s.ctx, "/etcd/Txn")
	defer tracing.FinishAnySpan(span)
	if span != nil {
		keys := make([]byte, 0, 512)
		for k := range s.wset {
			if len(keys) > 0 {
				keys = append(keys, ',')
			}
			keys = append(keys, []byte(k)...)
		}
		span.SetTag("updated-keys", string(keys))
	}

	keys, getops := s.gets()
	cmps := s.cmps()
	puts := s.puts()
	txn := s.client.Txn(ctx).If(cmps...).Then(puts...)
	// use Else to prefetch keys in case of conflict to save a round trip
	txnresp, err := txn.Else(getops...).Commit()
	if err == rpctypes.ErrTooManyOps {
		panic(stmError{
			fmt.Errorf(
				"%v (%d comparisons, %d puts: hint: set --max-txn-ops on the "+
					"ETCD cluster to at least the largest of those values)",
				err, len(cmps), len(puts)),
		})
	} else if err != nil {
		panic(stmError{err})
	}

	if span != nil {
		span.SetTag("applied-at-revision", txnresp.Header.Revision)
	}
	if txnresp.Succeeded {
		return txnresp
	}
	// load prefetch with Else data
	for i := range keys {
		resp := txnresp.Responses[i].GetResponseRange()
		s.rset[keys[i]] = (*v3.GetResponse)(resp)
	}
	s.prefetch = s.rset
	s.getOpts = nil
	return nil
}

type stmReadCommitted struct{ stm }

// commit always goes through when read committed
func (s *stmReadCommitted) commit() *v3.TxnResponse {
	s.rset = nil
	return s.stm.commit()
}

func isKeyCurrent(k string, r *v3.GetResponse) v3.Cmp {
	if len(r.Kvs) != 0 {
		return v3.Compare(v3.ModRevision(k), "=", r.Kvs[0].ModRevision)
	}
	return v3.Compare(v3.ModRevision(k), "=", 0)
}

func respToValue(key string, resp *v3.GetResponse) (string, error) {
	if len(resp.Kvs) == 0 {
		return "", ErrNotFound{Key: key}
	}
	return string(resp.Kvs[0].Value), nil
}

// fetchTTL contains the essential implementation of TTL().
//
// Note that 'iface' should either be the receiver 's' or a containing
// 'stmSerializeable'--the only reason 'iface' is passed as a separate argument
// is because fetchTTL calls iface.fetch(), and the implementation of 'fetch' is
// different for stm and stmSerializeable. Passing the interface ensures the
// correct version of fetch() is called
func (s *stm) fetchTTL(iface STM, key string) (int64, error) {
	// check wset cache
	if wv, ok := s.wset[key]; ok {
		return wv.ttl, nil
	}

	// Read ttl through s.ttlset cache
	if ttl, ok := s.ttlset[key]; ok {
		return ttl, nil
	}

	// Read kv and lease ID, and cache new TTL
	getResp := iface.fetch(key) // call correct implementation of fetch()
	if len(getResp.Kvs) == 0 {
		return 0, ErrNotFound{Key: key}
	}
	leaseID := v3.LeaseID(getResp.Kvs[0].Lease)
	if leaseID == 0 {
		s.ttlset[key] = 0 // 0 is default value, but now 'ok' will be true on check
		return 0, nil
	}
	span, ctx := tracing.AddSpanToAnyExisting(s.ctx, "/etcd/TimeToLive")
	defer tracing.FinishAnySpan(span)
	leaseResp, err := s.client.TimeToLive(ctx, leaseID)
	if err != nil {
		panic(stmError{err})
	}
	s.ttlset[key] = leaseResp.TTL
	for _, key := range leaseResp.Keys {
		s.ttlset[string(key)] = leaseResp.TTL
	}
	return leaseResp.TTL, nil
}

func (s *stm) TTL(key string) (int64, error) {
	return s.fetchTTL(s, key)
}

func (s *stmSerializable) TTL(key string) (int64, error) {
	return s.fetchTTL(s, key)
}
