// Code generated by protoc-gen-go. DO NOT EDIT.
// source: schema/schema.proto

package schema

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type MethodOptions struct {
	// Types that are valid to be assigned to Type:
	//	*MethodOptions_Query
	//	*MethodOptions_Mutation
	Type                 isMethodOptions_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *MethodOptions) Reset()         { *m = MethodOptions{} }
func (m *MethodOptions) String() string { return proto.CompactTextString(m) }
func (*MethodOptions) ProtoMessage()    {}
func (*MethodOptions) Descriptor() ([]byte, []int) {
	return fileDescriptor_98b0d2c3e7e0142d, []int{0}
}

func (m *MethodOptions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MethodOptions.Unmarshal(m, b)
}
func (m *MethodOptions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MethodOptions.Marshal(b, m, deterministic)
}
func (m *MethodOptions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MethodOptions.Merge(m, src)
}
func (m *MethodOptions) XXX_Size() int {
	return xxx_messageInfo_MethodOptions.Size(m)
}
func (m *MethodOptions) XXX_DiscardUnknown() {
	xxx_messageInfo_MethodOptions.DiscardUnknown(m)
}

var xxx_messageInfo_MethodOptions proto.InternalMessageInfo

type isMethodOptions_Type interface {
	isMethodOptions_Type()
}

type MethodOptions_Query struct {
	Query string `protobuf:"bytes,1,opt,name=query,proto3,oneof"`
}

type MethodOptions_Mutation struct {
	Mutation string `protobuf:"bytes,2,opt,name=mutation,proto3,oneof"`
}

func (*MethodOptions_Query) isMethodOptions_Type() {}

func (*MethodOptions_Mutation) isMethodOptions_Type() {}

func (m *MethodOptions) GetType() isMethodOptions_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *MethodOptions) GetQuery() string {
	if x, ok := m.GetType().(*MethodOptions_Query); ok {
		return x.Query
	}
	return ""
}

func (m *MethodOptions) GetMutation() string {
	if x, ok := m.GetType().(*MethodOptions_Mutation); ok {
		return x.Mutation
	}
	return ""
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*MethodOptions) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*MethodOptions_Query)(nil),
		(*MethodOptions_Mutation)(nil),
	}
}

var E_Schema = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MethodOptions)(nil),
	ExtensionType: (*MethodOptions)(nil),
	Field:         91111,
	Name:          "graphql.schema",
	Tag:           "bytes,91111,opt,name=schema",
	Filename:      "schema/schema.proto",
}

var E_Skip = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         91112,
	Name:          "graphql.skip",
	Tag:           "varint,91112,opt,name=skip",
	Filename:      "schema/schema.proto",
}

var E_Name = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         91114,
	Name:          "graphql.name",
	Tag:           "bytes,91114,opt,name=name",
	Filename:      "schema/schema.proto",
}

var E_Type = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.MessageOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         91117,
	Name:          "graphql.type",
	Tag:           "bytes,91117,opt,name=type",
	Filename:      "schema/schema.proto",
}

var E_FileSkip = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FileOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         91113,
	Name:          "graphql.file_skip",
	Tag:           "varint,91113,opt,name=file_skip",
	Filename:      "schema/schema.proto",
}

var E_InputSkip = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         91115,
	Name:          "graphql.input_skip",
	Tag:           "varint,91115,opt,name=input_skip",
	Filename:      "schema/schema.proto",
}

var E_PayloadSkip = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         91116,
	Name:          "graphql.payload_skip",
	Tag:           "varint,91116,opt,name=payload_skip",
	Filename:      "schema/schema.proto",
}

var E_Id = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         91120,
	Name:          "graphql.id",
	Tag:           "varint,91120,opt,name=id",
	Filename:      "schema/schema.proto",
}

var E_FieldName = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         91121,
	Name:          "graphql.field_name",
	Tag:           "bytes,91121,opt,name=field_name",
	Filename:      "schema/schema.proto",
}

func init() {
	proto.RegisterType((*MethodOptions)(nil), "graphql.MethodOptions")
	proto.RegisterExtension(E_Schema)
	proto.RegisterExtension(E_Skip)
	proto.RegisterExtension(E_Name)
	proto.RegisterExtension(E_Type)
	proto.RegisterExtension(E_FileSkip)
	proto.RegisterExtension(E_InputSkip)
	proto.RegisterExtension(E_PayloadSkip)
	proto.RegisterExtension(E_Id)
	proto.RegisterExtension(E_FieldName)
}

func init() { proto.RegisterFile("schema/schema.proto", fileDescriptor_98b0d2c3e7e0142d) }

var fileDescriptor_98b0d2c3e7e0142d = []byte{
	// 355 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0x4f, 0x4f, 0xf2, 0x40,
	0x10, 0x87, 0x5f, 0x08, 0xf0, 0xd2, 0x41, 0x2f, 0x35, 0x21, 0x44, 0x41, 0x89, 0x27, 0x4e, 0xdb,
	0x44, 0xe3, 0x05, 0x13, 0x0f, 0x1c, 0x8c, 0x17, 0xd4, 0xd4, 0x9b, 0x17, 0xb2, 0xd0, 0xa5, 0xac,
	0x6e, 0xbb, 0x4b, 0xbb, 0x3d, 0xf4, 0x13, 0xfa, 0x51, 0xfc, 0x9f, 0xe8, 0x37, 0x30, 0x3b, 0x5d,
	0x48, 0x88, 0x24, 0xf5, 0xd4, 0x74, 0x67, 0x9e, 0x67, 0x7e, 0x3b, 0x59, 0xd8, 0x4b, 0x67, 0x0b,
	0x16, 0x51, 0xaf, 0xf8, 0x10, 0x95, 0x48, 0x2d, 0xdd, 0xff, 0x61, 0x42, 0xd5, 0x62, 0x29, 0xf6,
	0xfb, 0xa1, 0x94, 0xa1, 0x60, 0x1e, 0x1e, 0x4f, 0xb3, 0xb9, 0x17, 0xb0, 0x74, 0x96, 0x70, 0xa5,
	0x65, 0x52, 0xb4, 0x1e, 0x8f, 0x61, 0x77, 0xcc, 0xf4, 0x42, 0x06, 0x37, 0x4a, 0x73, 0x19, 0xa7,
	0x6e, 0x1b, 0xea, 0xcb, 0x8c, 0x25, 0x79, 0xa7, 0xd2, 0xaf, 0x0c, 0x9c, 0xab, 0x7f, 0x7e, 0xf1,
	0xeb, 0x76, 0xa1, 0x19, 0x65, 0x9a, 0x9a, 0xa6, 0x4e, 0xd5, 0x96, 0xd6, 0x27, 0xa3, 0x06, 0xd4,
	0x74, 0xae, 0xd8, 0xf0, 0x16, 0x1a, 0x45, 0x12, 0xf7, 0x90, 0x14, 0xb3, 0xc9, 0x6a, 0x36, 0xd9,
	0x98, 0xd3, 0x79, 0x7e, 0xaa, 0xf7, 0x2b, 0x83, 0xd6, 0x49, 0x9b, 0xd8, 0xb0, 0x9b, 0x75, 0xdf,
	0x7a, 0x86, 0x67, 0x50, 0x4b, 0x1f, 0xb9, 0x72, 0x8f, 0xb6, 0xf8, 0xd2, 0x94, 0x86, 0x6c, 0x25,
	0x7c, 0x41, 0x61, 0xd3, 0xc7, 0x76, 0x83, 0xc5, 0x34, 0x62, 0xe5, 0xd8, 0x1b, 0x62, 0x8e, 0x8f,
	0xed, 0x06, 0x33, 0xf7, 0x28, 0xc7, 0x3e, 0x57, 0x18, 0x5e, 0xfb, 0x1c, 0x9c, 0x39, 0x17, 0x6c,
	0x82, 0x49, 0xbb, 0xbf, 0xd8, 0x4b, 0x2e, 0xd6, 0xe0, 0xab, 0x8d, 0xd9, 0x34, 0xc0, 0x9d, 0x89,
	0x7a, 0x01, 0xc0, 0x63, 0x95, 0xe9, 0x82, 0xee, 0x6d, 0xa1, 0x99, 0x58, 0xaf, 0xed, 0xdd, 0xe2,
	0x0e, 0x22, 0xc8, 0x8f, 0x60, 0x47, 0xd1, 0x5c, 0x48, 0x1a, 0xfc, 0xc9, 0xf0, 0x61, 0x0d, 0x2d,
	0x0b, 0xa1, 0xc3, 0x83, 0x2a, 0x0f, 0xca, 0xc8, 0x2f, 0x4b, 0x56, 0x79, 0x60, 0x42, 0xcf, 0x4d,
	0x6d, 0x82, 0x5b, 0x2e, 0x01, 0xbf, 0xed, 0xb2, 0x1c, 0x44, 0xae, 0x69, 0xc4, 0x46, 0xbd, 0xfb,
	0x83, 0x50, 0x12, 0xaa, 0x94, 0xe4, 0xb1, 0xce, 0xc9, 0x4c, 0x46, 0xde, 0x03, 0xa5, 0xc2, 0xbe,
	0xe3, 0x69, 0x03, 0x45, 0xa7, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x92, 0x49, 0x8a, 0xa4, 0xdf,
	0x02, 0x00, 0x00,
}
