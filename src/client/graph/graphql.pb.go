// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: client/graph/graphql.proto

package graph

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

var E_Graphql = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FileOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         64040,
	Name:          "graphqlproto.graphql",
	Tag:           "varint,64040,opt,name=graphql",
	Filename:      "client/graph/graphql.proto",
}

var E_Bitflags = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         64041,
	Name:          "graphqlproto.bitflags",
	Tag:           "varint,64041,opt,name=bitflags",
	Filename:      "client/graph/graphql.proto",
}

var E_Required = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         60000,
	Name:          "graphqlproto.required",
	Tag:           "varint,60000,opt,name=required",
	Filename:      "client/graph/graphql.proto",
}

func init() {
	proto.RegisterExtension(E_Graphql)
	proto.RegisterExtension(E_Bitflags)
	proto.RegisterExtension(E_Required)
}

func init() { proto.RegisterFile("client/graph/graphql.proto", fileDescriptor_a5c808080a6ff220) }

var fileDescriptor_a5c808080a6ff220 = []byte{
	// 254 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4a, 0xce, 0xc9, 0x4c,
	0xcd, 0x2b, 0xd1, 0x4f, 0x2f, 0x4a, 0x2c, 0xc8, 0x80, 0x90, 0x85, 0x39, 0x7a, 0x05, 0x45, 0xf9,
	0x25, 0xf9, 0x42, 0x3c, 0x50, 0x2e, 0x98, 0x27, 0xa5, 0x90, 0x9e, 0x9f, 0x9f, 0x9e, 0x93, 0xaa,
	0x0f, 0xe6, 0x25, 0x95, 0xa6, 0xe9, 0xa7, 0xa4, 0x16, 0x27, 0x17, 0x65, 0x16, 0x94, 0xe4, 0x17,
	0x41, 0xd4, 0x5b, 0x59, 0x70, 0xb1, 0x43, 0x75, 0x08, 0xc9, 0xe8, 0x41, 0x54, 0xeb, 0xc1, 0x54,
	0xeb, 0xb9, 0x65, 0xe6, 0xa4, 0xfa, 0x17, 0x94, 0x64, 0xe6, 0xe7, 0x15, 0x4b, 0xac, 0xf8, 0xc2,
	0xac, 0xc0, 0xa8, 0xc1, 0x11, 0x04, 0x53, 0x6e, 0x65, 0xcb, 0xc5, 0x91, 0x94, 0x59, 0x92, 0x96,
	0x93, 0x98, 0x5e, 0x2c, 0x24, 0x8f, 0xa1, 0xd5, 0x37, 0xb5, 0xb8, 0x38, 0x31, 0x1d, 0xae, 0x7b,
	0x25, 0x54, 0x37, 0x5c, 0x8b, 0x95, 0x35, 0x17, 0x47, 0x51, 0x6a, 0x61, 0x69, 0x66, 0x51, 0x6a,
	0x8a, 0x90, 0x2c, 0x16, 0x9b, 0x53, 0x73, 0x52, 0x60, 0x9a, 0x1f, 0x5c, 0x81, 0x6a, 0x86, 0x69,
	0x70, 0xca, 0xfe, 0xf1, 0x50, 0x8e, 0x71, 0xc5, 0x23, 0x39, 0xc6, 0x1d, 0x8f, 0xe4, 0x18, 0x4f,
	0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1, 0x23, 0x39, 0x46, 0x2e, 0xe1, 0xe4, 0xfc,
	0x5c, 0x74, 0xd3, 0x9c, 0xb8, 0xfd, 0x0b, 0x8a, 0x53, 0x53, 0x03, 0x40, 0xdc, 0xe2, 0x28, 0xfd,
	0xf4, 0xcc, 0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc, 0x5c, 0xfd, 0x82, 0xc4, 0xe4, 0x8c, 0xca,
	0x94, 0xd4, 0x22, 0x64, 0x56, 0x71, 0x51, 0xb2, 0x3e, 0x72, 0x00, 0x27, 0xb1, 0x81, 0xcd, 0x31,
	0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x23, 0x08, 0x43, 0xf9, 0x77, 0x01, 0x00, 0x00,
}
