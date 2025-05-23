// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v5.28.2
// source: protos/environment.proto

package bkpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EnvironmentCreationRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id               string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status           *string `protobuf:"bytes,3,opt,name=status,proto3,oneof" json:"status,omitempty"`
	StatusMessage    *string `protobuf:"bytes,4,opt,name=statusMessage,proto3,oneof" json:"statusMessage,omitempty"`
	RawConfiguration string  `protobuf:"bytes,5,opt,name=rawConfiguration,proto3" json:"rawConfiguration,omitempty"`
}

func (x *EnvironmentCreationRequest) Reset() {
	*x = EnvironmentCreationRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_environment_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentCreationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentCreationRequest) ProtoMessage() {}

func (x *EnvironmentCreationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_environment_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentCreationRequest.ProtoReflect.Descriptor instead.
func (*EnvironmentCreationRequest) Descriptor() ([]byte, []int) {
	return file_protos_environment_proto_rawDescGZIP(), []int{0}
}

func (x *EnvironmentCreationRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *EnvironmentCreationRequest) GetStatus() string {
	if x != nil && x.Status != nil {
		return *x.Status
	}
	return ""
}

func (x *EnvironmentCreationRequest) GetStatusMessage() string {
	if x != nil && x.StatusMessage != nil {
		return *x.StatusMessage
	}
	return ""
}

func (x *EnvironmentCreationRequest) GetRawConfiguration() string {
	if x != nil {
		return x.RawConfiguration
	}
	return ""
}

type EnvironmentUpdateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status        *string `protobuf:"bytes,3,opt,name=status,proto3,oneof" json:"status,omitempty"`
	StatusMessage *string `protobuf:"bytes,4,opt,name=statusMessage,proto3,oneof" json:"statusMessage,omitempty"`
}

func (x *EnvironmentUpdateRequest) Reset() {
	*x = EnvironmentUpdateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_environment_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentUpdateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentUpdateRequest) ProtoMessage() {}

func (x *EnvironmentUpdateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_environment_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentUpdateRequest.ProtoReflect.Descriptor instead.
func (*EnvironmentUpdateRequest) Descriptor() ([]byte, []int) {
	return file_protos_environment_proto_rawDescGZIP(), []int{1}
}

func (x *EnvironmentUpdateRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *EnvironmentUpdateRequest) GetStatus() string {
	if x != nil && x.Status != nil {
		return *x.Status
	}
	return ""
}

func (x *EnvironmentUpdateRequest) GetStatusMessage() string {
	if x != nil && x.StatusMessage != nil {
		return *x.StatusMessage
	}
	return ""
}

type Environment struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Unix timestamp when this entity was created.
	CreatedAt int64 `protobuf:"varint,2,opt,name=createdAt,proto3" json:"createdAt,omitempty"`
	// Unix timestamp when this entity was last updated.
	UpdatedAt int64 `protobuf:"varint,3,opt,name=updatedAt,proto3" json:"updatedAt,omitempty"`
	// The status of the environment.
	Status string `protobuf:"bytes,5,opt,name=status,proto3" json:"status,omitempty"`
	// A message explaining the status or the current state of the environment.
	StatusMessage string `protobuf:"bytes,6,opt,name=statusMessage,proto3" json:"statusMessage,omitempty"`
	// Array of minified Run objects.
	Runs []*Run `protobuf:"bytes,7,rep,name=runs,proto3" json:"runs,omitempty"`
}

func (x *Environment) Reset() {
	*x = Environment{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_environment_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Environment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Environment) ProtoMessage() {}

func (x *Environment) ProtoReflect() protoreflect.Message {
	mi := &file_protos_environment_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Environment.ProtoReflect.Descriptor instead.
func (*Environment) Descriptor() ([]byte, []int) {
	return file_protos_environment_proto_rawDescGZIP(), []int{2}
}

func (x *Environment) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Environment) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

func (x *Environment) GetUpdatedAt() int64 {
	if x != nil {
		return x.UpdatedAt
	}
	return 0
}

func (x *Environment) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *Environment) GetStatusMessage() string {
	if x != nil {
		return x.StatusMessage
	}
	return ""
}

func (x *Environment) GetRuns() []*Run {
	if x != nil {
		return x.Runs
	}
	return nil
}

var File_protos_environment_proto protoreflect.FileDescriptor

var file_protos_environment_proto_rawDesc = []byte{
	0x0a, 0x18, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x6f, 0x32, 0x2e, 0x62,
	0x6f, 0x6f, 0x6b, 0x6b, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67, 0x1a, 0x15, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x2f, 0x62, 0x6b, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xbd, 0x01, 0x0a, 0x1a, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64,
	0x12, 0x1b, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x48, 0x00, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x88, 0x01, 0x01, 0x12, 0x29, 0x0a,
	0x0d, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x0d, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x88, 0x01, 0x01, 0x12, 0x2a, 0x0a, 0x10, 0x72, 0x61, 0x77, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x10, 0x72, 0x61, 0x77, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x42, 0x09, 0x0a, 0x07, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x42,
	0x10, 0x0a, 0x0e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x22, 0x8f, 0x01, 0x0a, 0x18, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1b,
	0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x88, 0x01, 0x01, 0x12, 0x29, 0x0a, 0x0d, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x01, 0x52, 0x0d, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x88, 0x01, 0x01, 0x42, 0x09, 0x0a, 0x07, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x42, 0x10, 0x0a, 0x0e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x22, 0xc0, 0x01, 0x0a, 0x0b, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41,
	0x74, 0x12, 0x1c, 0x0a, 0x09, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12,
	0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x24, 0x0a, 0x0d, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d,
	0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x27, 0x0a,
	0x04, 0x72, 0x75, 0x6e, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6f, 0x32,
	0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6b, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67, 0x2e, 0x52, 0x75, 0x6e,
	0x52, 0x04, 0x72, 0x75, 0x6e, 0x73, 0x32, 0xb8, 0x01, 0x0a, 0x12, 0x45, 0x6e, 0x76, 0x69, 0x72,
	0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x51, 0x0a,
	0x06, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x12, 0x2a, 0x2e, 0x6f, 0x32, 0x2e, 0x62, 0x6f, 0x6f,
	0x6b, 0x6b, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x43, 0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x6f, 0x32, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6b, 0x65, 0x65,
	0x70, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74,
	0x12, 0x4f, 0x0a, 0x06, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x28, 0x2e, 0x6f, 0x32, 0x2e,
	0x62, 0x6f, 0x6f, 0x6b, 0x6b, 0x65, 0x65, 0x70, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x6e, 0x76, 0x69,
	0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x6f, 0x32, 0x2e, 0x62, 0x6f, 0x6f, 0x6b, 0x6b, 0x65,
	0x65, 0x70, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x42, 0x4a, 0x5a, 0x48, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x41, 0x6c, 0x69, 0x63, 0x65, 0x4f, 0x32, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x2f, 0x43, 0x6f, 0x6e,
	0x74, 0x72, 0x6f, 0x6c, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x62, 0x6f, 0x6f, 0x6b, 0x6b, 0x65, 0x65, 0x70, 0x69, 0x6e,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x3b, 0x62, 0x6b, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protos_environment_proto_rawDescOnce sync.Once
	file_protos_environment_proto_rawDescData = file_protos_environment_proto_rawDesc
)

func file_protos_environment_proto_rawDescGZIP() []byte {
	file_protos_environment_proto_rawDescOnce.Do(func() {
		file_protos_environment_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_environment_proto_rawDescData)
	})
	return file_protos_environment_proto_rawDescData
}

var file_protos_environment_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_protos_environment_proto_goTypes = []interface{}{
	(*EnvironmentCreationRequest)(nil), // 0: o2.bookkeeping.EnvironmentCreationRequest
	(*EnvironmentUpdateRequest)(nil),   // 1: o2.bookkeeping.EnvironmentUpdateRequest
	(*Environment)(nil),                // 2: o2.bookkeeping.Environment
	(*Run)(nil),                        // 3: o2.bookkeeping.Run
}
var file_protos_environment_proto_depIdxs = []int32{
	3, // 0: o2.bookkeeping.Environment.runs:type_name -> o2.bookkeeping.Run
	0, // 1: o2.bookkeeping.EnvironmentService.Create:input_type -> o2.bookkeeping.EnvironmentCreationRequest
	1, // 2: o2.bookkeeping.EnvironmentService.Update:input_type -> o2.bookkeeping.EnvironmentUpdateRequest
	2, // 3: o2.bookkeeping.EnvironmentService.Create:output_type -> o2.bookkeeping.Environment
	2, // 4: o2.bookkeeping.EnvironmentService.Update:output_type -> o2.bookkeeping.Environment
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_protos_environment_proto_init() }
func file_protos_environment_proto_init() {
	if File_protos_environment_proto != nil {
		return
	}
	file_protos_bkcommon_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_protos_environment_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentCreationRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_protos_environment_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentUpdateRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_protos_environment_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Environment); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_protos_environment_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_protos_environment_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_protos_environment_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protos_environment_proto_goTypes,
		DependencyIndexes: file_protos_environment_proto_depIdxs,
		MessageInfos:      file_protos_environment_proto_msgTypes,
	}.Build()
	File_protos_environment_proto = out.File
	file_protos_environment_proto_rawDesc = nil
	file_protos_environment_proto_goTypes = nil
	file_protos_environment_proto_depIdxs = nil
}
