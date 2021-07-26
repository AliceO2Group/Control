// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.17.3
// source: protos/ddsched.proto

// Changelog:
//      2021-02-08: - Initial version
//                  - Support for a single partition/environment: PartitionID = EnvironmentID

package ddpb

import (
	proto "github.com/golang/protobuf/proto"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type PartitionState int32

const (
	PartitionState_IGNORE__                  PartitionState = 0
	PartitionState_PARTITION_UNKNOWN         PartitionState = 1 // Partition not known to DataDistControl. Initialize by calling PartitionInitialize()
	PartitionState_PARTITION_ERROR           PartitionState = 2
	PartitionState_PARTITION_REQUEST_INVALID PartitionState = 3 // Provided configuration does not meet requirements or the request is not accepted at the current state.
	PartitionState_PARTITION_CONFIGURING     PartitionState = 4 // Configuration accepted, waiting on FLP and EPN DD components to respond and connect.
	PartitionState_PARTITION_CONFIGURED      PartitionState = 5 // All components configured, ready for dataflow commands
	PartitionState_PARTITION_TERMINATING     PartitionState = 6 // Partition is terminating. EPN-FLP connections will be cleanly closed.
	PartitionState_PARTITION_TERMINATED      PartitionState = 7 // Partition is terminated. TfScheduler does not accept further requests
)

// Enum value maps for PartitionState.
var (
	PartitionState_name = map[int32]string{
		0: "IGNORE__",
		1: "PARTITION_UNKNOWN",
		2: "PARTITION_ERROR",
		3: "PARTITION_REQUEST_INVALID",
		4: "PARTITION_CONFIGURING",
		5: "PARTITION_CONFIGURED",
		6: "PARTITION_TERMINATING",
		7: "PARTITION_TERMINATED",
	}
	PartitionState_value = map[string]int32{
		"IGNORE__":                  0,
		"PARTITION_UNKNOWN":         1,
		"PARTITION_ERROR":           2,
		"PARTITION_REQUEST_INVALID": 3,
		"PARTITION_CONFIGURING":     4,
		"PARTITION_CONFIGURED":      5,
		"PARTITION_TERMINATING":     6,
		"PARTITION_TERMINATED":      7,
	}
)

func (x PartitionState) Enum() *PartitionState {
	p := new(PartitionState)
	*p = x
	return p
}

func (x PartitionState) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PartitionState) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_ddsched_proto_enumTypes[0].Descriptor()
}

func (PartitionState) Type() protoreflect.EnumType {
	return &file_protos_ddsched_proto_enumTypes[0]
}

func (x PartitionState) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PartitionState.Descriptor instead.
func (PartitionState) EnumDescriptor() ([]byte, []int) {
	return file_protos_ddsched_proto_rawDescGZIP(), []int{0}
}

type PartitionInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	EnvironmentId string `protobuf:"bytes,1,opt,name=environment_id,json=environmentId,proto3" json:"environment_id,omitempty"`
	// AliECS environment ID (required)
	PartitionId string `protobuf:"bytes,2,opt,name=partition_id,json=partitionId,proto3" json:"partition_id,omitempty"` // Partition ID. (required)
}

func (x *PartitionInfo) Reset() {
	*x = PartitionInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_ddsched_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PartitionInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PartitionInfo) ProtoMessage() {}

func (x *PartitionInfo) ProtoReflect() protoreflect.Message {
	mi := &file_protos_ddsched_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PartitionInfo.ProtoReflect.Descriptor instead.
func (*PartitionInfo) Descriptor() ([]byte, []int) {
	return file_protos_ddsched_proto_rawDescGZIP(), []int{0}
}

func (x *PartitionInfo) GetEnvironmentId() string {
	if x != nil {
		return x.EnvironmentId
	}
	return ""
}

func (x *PartitionInfo) GetPartitionId() string {
	if x != nil {
		return x.PartitionId
	}
	return ""
}

type PartitionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PartitionState PartitionState `protobuf:"varint,1,opt,name=partition_state,json=partitionState,proto3,enum=o2.DataDistribution.Control.PartitionState" json:"partition_state,omitempty"` // Current or new state of the partition following reception of the request.
	InfoMessage    string         `protobuf:"bytes,2,opt,name=info_message,json=infoMessage,proto3" json:"info_message,omitempty"`                                                           // Optional information message.
}

func (x *PartitionResponse) Reset() {
	*x = PartitionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_ddsched_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PartitionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PartitionResponse) ProtoMessage() {}

func (x *PartitionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protos_ddsched_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PartitionResponse.ProtoReflect.Descriptor instead.
func (*PartitionResponse) Descriptor() ([]byte, []int) {
	return file_protos_ddsched_proto_rawDescGZIP(), []int{1}
}

func (x *PartitionResponse) GetPartitionState() PartitionState {
	if x != nil {
		return x.PartitionState
	}
	return PartitionState_IGNORE__
}

func (x *PartitionResponse) GetInfoMessage() string {
	if x != nil {
		return x.InfoMessage
	}
	return ""
}

// Request for a new partition.
// AliECS provides the information about processes of DD in order for partition to be configured by TfScheduler.
// Partition request must include all processes. Adding additional processes to existing partition is not supported.
type PartitionInitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PartitionInfo *PartitionInfo    `protobuf:"bytes,1,opt,name=partition_info,json=partitionInfo,proto3" json:"partition_info,omitempty"`
	StfbHostIdMap map[string]string `protobuf:"bytes,2,rep,name=stfb_host_id_map,json=stfbHostIdMap,proto3" json:"stfb_host_id_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Mapping between StfBuilder::discovery-id parameter and its FLP host name (must resolve to IB IP), for all
	// StfBuilder processes in the partition.
	StfsHostIdMap map[string]string `protobuf:"bytes,3,rep,name=stfs_host_id_map,json=stfsHostIdMap,proto3" json:"stfs_host_id_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *PartitionInitRequest) Reset() {
	*x = PartitionInitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_ddsched_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PartitionInitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PartitionInitRequest) ProtoMessage() {}

func (x *PartitionInitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_ddsched_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PartitionInitRequest.ProtoReflect.Descriptor instead.
func (*PartitionInitRequest) Descriptor() ([]byte, []int) {
	return file_protos_ddsched_proto_rawDescGZIP(), []int{2}
}

func (x *PartitionInitRequest) GetPartitionInfo() *PartitionInfo {
	if x != nil {
		return x.PartitionInfo
	}
	return nil
}

func (x *PartitionInitRequest) GetStfbHostIdMap() map[string]string {
	if x != nil {
		return x.StfbHostIdMap
	}
	return nil
}

func (x *PartitionInitRequest) GetStfsHostIdMap() map[string]string {
	if x != nil {
		return x.StfsHostIdMap
	}
	return nil
}

// Request for partition to be terminated.
// This operation will terminate all connections between EPN and FLP, and remove the partition from DD control-plane.
// Lifetime of individual DD components is managed by the respective FairMQ controller.
type PartitionTermRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PartitionInfo *PartitionInfo `protobuf:"bytes,1,opt,name=partition_info,json=partitionInfo,proto3" json:"partition_info,omitempty"`
}

func (x *PartitionTermRequest) Reset() {
	*x = PartitionTermRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_ddsched_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PartitionTermRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PartitionTermRequest) ProtoMessage() {}

func (x *PartitionTermRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_ddsched_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PartitionTermRequest.ProtoReflect.Descriptor instead.
func (*PartitionTermRequest) Descriptor() ([]byte, []int) {
	return file_protos_ddsched_proto_rawDescGZIP(), []int{3}
}

func (x *PartitionTermRequest) GetPartitionInfo() *PartitionInfo {
	if x != nil {
		return x.PartitionInfo
	}
	return nil
}

var File_protos_ddsched_proto protoreflect.FileDescriptor

var file_protos_ddsched_proto_rawDesc = []byte{
	0x0a, 0x14, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x64, 0x64, 0x73, 0x63, 0x68, 0x65, 0x64,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44,
	0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74,
	0x72, 0x6f, 0x6c, 0x22, 0x59, 0x0a, 0x0d, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x49, 0x6e, 0x66, 0x6f, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x65, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x70,
	0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x22, 0x8c,
	0x01, 0x0a, 0x11, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x54, 0x0a, 0x0f, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x2b, 0x2e,
	0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72, 0x74,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x0e, 0x70, 0x61, 0x72, 0x74,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x69, 0x6e,
	0x66, 0x6f, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x69, 0x6e, 0x66, 0x6f, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0xcb, 0x03,
	0x0a, 0x14, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x69, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x51, 0x0a, 0x0e, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a,
	0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72,
	0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0d, 0x70, 0x61, 0x72, 0x74,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x6d, 0x0a, 0x10, 0x73, 0x74, 0x66,
	0x62, 0x5f, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x44, 0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73,
	0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f,
	0x6c, 0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x69, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x66, 0x62, 0x48, 0x6f, 0x73, 0x74, 0x49,
	0x64, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0d, 0x73, 0x74, 0x66, 0x62, 0x48,
	0x6f, 0x73, 0x74, 0x49, 0x64, 0x4d, 0x61, 0x70, 0x12, 0x6d, 0x0a, 0x10, 0x73, 0x74, 0x66, 0x73,
	0x5f, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x44, 0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x66, 0x73, 0x48, 0x6f, 0x73, 0x74, 0x49, 0x64,
	0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0d, 0x73, 0x74, 0x66, 0x73, 0x48, 0x6f,
	0x73, 0x74, 0x49, 0x64, 0x4d, 0x61, 0x70, 0x1a, 0x40, 0x0a, 0x12, 0x53, 0x74, 0x66, 0x62, 0x48,
	0x6f, 0x73, 0x74, 0x49, 0x64, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x40, 0x0a, 0x12, 0x53, 0x74, 0x66,
	0x73, 0x48, 0x6f, 0x73, 0x74, 0x49, 0x64, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x69, 0x0a, 0x14, 0x50,
	0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x65, 0x72, 0x6d, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x51, 0x0a, 0x0e, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x6f, 0x32,
	0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f,
	0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0d, 0x70, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x2a, 0xd3, 0x01, 0x0a, 0x0e, 0x50, 0x61, 0x72, 0x74, 0x69,
	0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x0c, 0x0a, 0x08, 0x49, 0x47, 0x4e,
	0x4f, 0x52, 0x45, 0x5f, 0x5f, 0x10, 0x00, 0x12, 0x15, 0x0a, 0x11, 0x50, 0x41, 0x52, 0x54, 0x49,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x01, 0x12, 0x13,
	0x0a, 0x0f, 0x50, 0x41, 0x52, 0x54, 0x49, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x45, 0x52, 0x52, 0x4f,
	0x52, 0x10, 0x02, 0x12, 0x1d, 0x0a, 0x19, 0x50, 0x41, 0x52, 0x54, 0x49, 0x54, 0x49, 0x4f, 0x4e,
	0x5f, 0x52, 0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x5f, 0x49, 0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44,
	0x10, 0x03, 0x12, 0x19, 0x0a, 0x15, 0x50, 0x41, 0x52, 0x54, 0x49, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x43, 0x4f, 0x4e, 0x46, 0x49, 0x47, 0x55, 0x52, 0x49, 0x4e, 0x47, 0x10, 0x04, 0x12, 0x18, 0x0a,
	0x14, 0x50, 0x41, 0x52, 0x54, 0x49, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x43, 0x4f, 0x4e, 0x46, 0x49,
	0x47, 0x55, 0x52, 0x45, 0x44, 0x10, 0x05, 0x12, 0x19, 0x0a, 0x15, 0x50, 0x41, 0x52, 0x54, 0x49,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x54, 0x45, 0x52, 0x4d, 0x49, 0x4e, 0x41, 0x54, 0x49, 0x4e, 0x47,
	0x10, 0x06, 0x12, 0x18, 0x0a, 0x14, 0x50, 0x41, 0x52, 0x54, 0x49, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x54, 0x45, 0x52, 0x4d, 0x49, 0x4e, 0x41, 0x54, 0x45, 0x44, 0x10, 0x07, 0x32, 0x81, 0x03, 0x0a,
	0x17, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f,
	0x6e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x12, 0x7a, 0x0a, 0x13, 0x50, 0x61, 0x72, 0x74,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x69, 0x74, 0x69, 0x61, 0x6c, 0x69, 0x7a, 0x65, 0x12,
	0x31, 0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62,
	0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61,
	0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x12, 0x79, 0x0a, 0x12, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x54, 0x65, 0x72, 0x6d, 0x69, 0x6e, 0x61, 0x74, 0x65, 0x12, 0x31, 0x2e, 0x6f, 0x32, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x54, 0x65, 0x72, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e,
	0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72, 0x74,
	0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12,
	0x6f, 0x0a, 0x0f, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x2a, 0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74,
	0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x2e, 0x50, 0x61, 0x72, 0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x1a, 0x2e,
	0x2e, 0x6f, 0x32, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x50, 0x61, 0x72,
	0x74, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00,
	0x42, 0x0d, 0x5a, 0x0b, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x3b, 0x64, 0x64, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protos_ddsched_proto_rawDescOnce sync.Once
	file_protos_ddsched_proto_rawDescData = file_protos_ddsched_proto_rawDesc
)

func file_protos_ddsched_proto_rawDescGZIP() []byte {
	file_protos_ddsched_proto_rawDescOnce.Do(func() {
		file_protos_ddsched_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_ddsched_proto_rawDescData)
	})
	return file_protos_ddsched_proto_rawDescData
}

var file_protos_ddsched_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_protos_ddsched_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_protos_ddsched_proto_goTypes = []interface{}{
	(PartitionState)(0),          // 0: o2.DataDistribution.Control.PartitionState
	(*PartitionInfo)(nil),        // 1: o2.DataDistribution.Control.PartitionInfo
	(*PartitionResponse)(nil),    // 2: o2.DataDistribution.Control.PartitionResponse
	(*PartitionInitRequest)(nil), // 3: o2.DataDistribution.Control.PartitionInitRequest
	(*PartitionTermRequest)(nil), // 4: o2.DataDistribution.Control.PartitionTermRequest
	nil,                          // 5: o2.DataDistribution.Control.PartitionInitRequest.StfbHostIdMapEntry
	nil,                          // 6: o2.DataDistribution.Control.PartitionInitRequest.StfsHostIdMapEntry
}
var file_protos_ddsched_proto_depIdxs = []int32{
	0, // 0: o2.DataDistribution.Control.PartitionResponse.partition_state:type_name -> o2.DataDistribution.Control.PartitionState
	1, // 1: o2.DataDistribution.Control.PartitionInitRequest.partition_info:type_name -> o2.DataDistribution.Control.PartitionInfo
	5, // 2: o2.DataDistribution.Control.PartitionInitRequest.stfb_host_id_map:type_name -> o2.DataDistribution.Control.PartitionInitRequest.StfbHostIdMapEntry
	6, // 3: o2.DataDistribution.Control.PartitionInitRequest.stfs_host_id_map:type_name -> o2.DataDistribution.Control.PartitionInitRequest.StfsHostIdMapEntry
	1, // 4: o2.DataDistribution.Control.PartitionTermRequest.partition_info:type_name -> o2.DataDistribution.Control.PartitionInfo
	3, // 5: o2.DataDistribution.Control.DataDistributionControl.PartitionInitialize:input_type -> o2.DataDistribution.Control.PartitionInitRequest
	4, // 6: o2.DataDistribution.Control.DataDistributionControl.PartitionTerminate:input_type -> o2.DataDistribution.Control.PartitionTermRequest
	1, // 7: o2.DataDistribution.Control.DataDistributionControl.PartitionStatus:input_type -> o2.DataDistribution.Control.PartitionInfo
	2, // 8: o2.DataDistribution.Control.DataDistributionControl.PartitionInitialize:output_type -> o2.DataDistribution.Control.PartitionResponse
	2, // 9: o2.DataDistribution.Control.DataDistributionControl.PartitionTerminate:output_type -> o2.DataDistribution.Control.PartitionResponse
	2, // 10: o2.DataDistribution.Control.DataDistributionControl.PartitionStatus:output_type -> o2.DataDistribution.Control.PartitionResponse
	8, // [8:11] is the sub-list for method output_type
	5, // [5:8] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_protos_ddsched_proto_init() }
func file_protos_ddsched_proto_init() {
	if File_protos_ddsched_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protos_ddsched_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PartitionInfo); i {
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
		file_protos_ddsched_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PartitionResponse); i {
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
		file_protos_ddsched_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PartitionInitRequest); i {
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
		file_protos_ddsched_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PartitionTermRequest); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_protos_ddsched_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protos_ddsched_proto_goTypes,
		DependencyIndexes: file_protos_ddsched_proto_depIdxs,
		EnumInfos:         file_protos_ddsched_proto_enumTypes,
		MessageInfos:      file_protos_ddsched_proto_msgTypes,
	}.Build()
	File_protos_ddsched_proto = out.File
	file_protos_ddsched_proto_rawDesc = nil
	file_protos_ddsched_proto_goTypes = nil
	file_protos_ddsched_proto_depIdxs = nil
}
