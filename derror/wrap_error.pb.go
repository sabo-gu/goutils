// Code generated by protoc-gen-go. DO NOT EDIT.
// source: wrap_error.proto

/*
Package derror is a generated protocol buffer package.

It is generated from these files:
	wrap_error.proto

It has these top-level messages:
	WrapError
*/
package derror

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ErrorLevel int32

const (
	ErrorLevel_Common  ErrorLevel = 0
	ErrorLevel_Serious ErrorLevel = 2012
)

var ErrorLevel_name = map[int32]string{
	0:    "Common",
	2012: "Serious",
}
var ErrorLevel_value = map[string]int32{
	"Common":  0,
	"Serious": 2012,
}

func (x ErrorLevel) String() string {
	return proto.EnumName(ErrorLevel_name, int32(x))
}
func (ErrorLevel) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type WrapError struct {
	Message         string     `protobuf:"bytes,1,opt,name=Message" json:"Message,omitempty"`
	Code            uint32     `protobuf:"varint,2,opt,name=Code" json:"Code,omitempty"`
	Unack           bool       `protobuf:"varint,4,opt,name=Unack" json:"Unack,omitempty"`
	Tips            bool       `protobuf:"varint,3,opt,name=Tips" json:"Tips,omitempty"`
	FriendlyMessage string     `protobuf:"bytes,5,opt,name=FriendlyMessage" json:"FriendlyMessage,omitempty"`
	Level           ErrorLevel `protobuf:"varint,6,opt,name=Level,enum=derror.ErrorLevel" json:"Level,omitempty"`
}

func (m *WrapError) Reset()                    { *m = WrapError{} }
func (m *WrapError) String() string            { return proto.CompactTextString(m) }
func (*WrapError) ProtoMessage()               {}
func (*WrapError) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *WrapError) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *WrapError) GetCode() uint32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *WrapError) GetUnack() bool {
	if m != nil {
		return m.Unack
	}
	return false
}

func (m *WrapError) GetTips() bool {
	if m != nil {
		return m.Tips
	}
	return false
}

func (m *WrapError) GetFriendlyMessage() string {
	if m != nil {
		return m.FriendlyMessage
	}
	return ""
}

func (m *WrapError) GetLevel() ErrorLevel {
	if m != nil {
		return m.Level
	}
	return ErrorLevel_Common
}

func init() {
	proto.RegisterType((*WrapError)(nil), "derror.WrapError")
	proto.RegisterEnum("derror.ErrorLevel", ErrorLevel_name, ErrorLevel_value)
}

func init() { proto.RegisterFile("wrap_error.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 209 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x28, 0x2f, 0x4a, 0x2c,
	0x88, 0x4f, 0x2d, 0x2a, 0xca, 0x2f, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4b, 0x01,
	0xf3, 0x94, 0xb6, 0x33, 0x72, 0x71, 0x86, 0x17, 0x25, 0x16, 0xb8, 0x82, 0x78, 0x42, 0x12, 0x5c,
	0xec, 0xbe, 0xa9, 0xc5, 0xc5, 0x89, 0xe9, 0xa9, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x30,
	0xae, 0x90, 0x10, 0x17, 0x8b, 0x73, 0x7e, 0x4a, 0xaa, 0x04, 0x93, 0x02, 0xa3, 0x06, 0x6f, 0x10,
	0x98, 0x2d, 0x24, 0xc2, 0xc5, 0x1a, 0x9a, 0x97, 0x98, 0x9c, 0x2d, 0xc1, 0xa2, 0xc0, 0xa8, 0xc1,
	0x11, 0x04, 0xe1, 0x80, 0x54, 0x86, 0x64, 0x16, 0x14, 0x4b, 0x30, 0x83, 0x05, 0xc1, 0x6c, 0x21,
	0x0d, 0x2e, 0x7e, 0xb7, 0xa2, 0xcc, 0xd4, 0xbc, 0x94, 0x9c, 0x4a, 0x98, 0xf9, 0xac, 0x60, 0xf3,
	0xd1, 0x85, 0x85, 0x34, 0xb8, 0x58, 0x7d, 0x52, 0xcb, 0x52, 0x73, 0x24, 0xd8, 0x14, 0x18, 0x35,
	0xf8, 0x8c, 0x84, 0xf4, 0x20, 0xee, 0xd4, 0x03, 0xbb, 0x0f, 0x2c, 0x13, 0x04, 0x51, 0xa0, 0xa5,
	0xc6, 0xc5, 0x85, 0x10, 0x14, 0xe2, 0xe2, 0x62, 0x73, 0xce, 0xcf, 0xcd, 0xcd, 0xcf, 0x13, 0x60,
	0x10, 0xe2, 0xe1, 0x62, 0x0f, 0x4e, 0x2d, 0xca, 0xcc, 0x2f, 0x2d, 0x16, 0xb8, 0xc3, 0x9f, 0xc4,
	0x06, 0xf6, 0xb0, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xa4, 0xa6, 0x07, 0x6d, 0x04, 0x01, 0x00,
	0x00,
}
