//
// === This file is part of ALICE O² ===
//
// Copyright 2020 CERN and copyright holders of ALICE O².
// Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// In applying this license CERN does not waive the privileges and
// immunities granted to it by virtue of its status as an
// Intergovernmental Organization or submit itself to any jurisdiction.

// Modified OmbrettaPinazza@DCS 13/11/2020

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.15.6
// source: protos/dcs.proto

package dcspb

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

type EventType int32

const (
	EventType_NULL_EVENT         EventType = 0
	EventType_HEARTBEAT          EventType = 1
	EventType_STATE_CHANGE_EVENT EventType = 2
	EventType_ERROR_EVENT        EventType = 3
	EventType_ACK_EVENT          EventType = 4
	EventType_SOR_EVENT          EventType = 20
	EventType_EOR_EVENT          EventType = 30
)

// Enum value maps for EventType.
var (
	EventType_name = map[int32]string{
		0:  "NULL_EVENT",
		1:  "HEARTBEAT",
		2:  "STATE_CHANGE_EVENT",
		3:  "ERROR_EVENT",
		4:  "ACK_EVENT",
		20: "SOR_EVENT",
		30: "EOR_EVENT",
	}
	EventType_value = map[string]int32{
		"NULL_EVENT":         0,
		"HEARTBEAT":          1,
		"STATE_CHANGE_EVENT": 2,
		"ERROR_EVENT":        3,
		"ACK_EVENT":          4,
		"SOR_EVENT":          20,
		"EOR_EVENT":          30,
	}
)

func (x EventType) Enum() *EventType {
	p := new(EventType)
	*p = x
	return p
}

func (x EventType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EventType) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_dcs_proto_enumTypes[0].Descriptor()
}

func (EventType) Type() protoreflect.EnumType {
	return &file_protos_dcs_proto_enumTypes[0]
}

func (x EventType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EventType.Descriptor instead.
func (EventType) EnumDescriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{0}
}

type Detector int32

const (
	Detector_NULL_DETECTOR Detector = 0
	Detector_CPV           Detector = 1
	Detector_EMC           Detector = 2
	Detector_FDD           Detector = 3
	Detector_FT0           Detector = 4
	Detector_FV0           Detector = 5
	Detector_ITS           Detector = 6
	Detector_HMP           Detector = 7
	Detector_MCH           Detector = 8
	Detector_MFT           Detector = 9
	Detector_MID           Detector = 10
	Detector_PHS           Detector = 11
	Detector_TOF           Detector = 12
	Detector_TPC           Detector = 13
	Detector_TRD           Detector = 14
	Detector_ZDC           Detector = 15
	Detector_DCS           Detector = 16
)

// Enum value maps for Detector.
var (
	Detector_name = map[int32]string{
		0:  "NULL_DETECTOR",
		1:  "CPV",
		2:  "EMC",
		3:  "FDD",
		4:  "FT0",
		5:  "FV0",
		6:  "ITS",
		7:  "HMP",
		8:  "MCH",
		9:  "MFT",
		10: "MID",
		11: "PHS",
		12: "TOF",
		13: "TPC",
		14: "TRD",
		15: "ZDC",
		16: "DCS",
	}
	Detector_value = map[string]int32{
		"NULL_DETECTOR": 0,
		"CPV":           1,
		"EMC":           2,
		"FDD":           3,
		"FT0":           4,
		"FV0":           5,
		"ITS":           6,
		"HMP":           7,
		"MCH":           8,
		"MFT":           9,
		"MID":           10,
		"PHS":           11,
		"TOF":           12,
		"TPC":           13,
		"TRD":           14,
		"ZDC":           15,
		"DCS":           16,
	}
)

func (x Detector) Enum() *Detector {
	p := new(Detector)
	*p = x
	return p
}

func (x Detector) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Detector) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_dcs_proto_enumTypes[1].Descriptor()
}

func (Detector) Type() protoreflect.EnumType {
	return &file_protos_dcs_proto_enumTypes[1]
}

func (x Detector) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Detector.Descriptor instead.
func (Detector) EnumDescriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{1}
}

type DetectorState int32

const (
	DetectorState_NULL_STATE      DetectorState = 0
	DetectorState_READY           DetectorState = 1
	DetectorState_RUN_OK          DetectorState = 2
	DetectorState_RUN_FAILURE     DetectorState = 3
	DetectorState_SOR_PROGRESSING DetectorState = 4
	DetectorState_EOR_PROGRESSING DetectorState = 5
	DetectorState_SOR_FAILURE     DetectorState = 6
	DetectorState_EOR_FAILURE     DetectorState = 7
	DetectorState_ERROR           DetectorState = 8
)

// Enum value maps for DetectorState.
var (
	DetectorState_name = map[int32]string{
		0: "NULL_STATE",
		1: "READY",
		2: "RUN_OK",
		3: "RUN_FAILURE",
		4: "SOR_PROGRESSING",
		5: "EOR_PROGRESSING",
		6: "SOR_FAILURE",
		7: "EOR_FAILURE",
		8: "ERROR",
	}
	DetectorState_value = map[string]int32{
		"NULL_STATE":      0,
		"READY":           1,
		"RUN_OK":          2,
		"RUN_FAILURE":     3,
		"SOR_PROGRESSING": 4,
		"EOR_PROGRESSING": 5,
		"SOR_FAILURE":     6,
		"EOR_FAILURE":     7,
		"ERROR":           8,
	}
)

func (x DetectorState) Enum() *DetectorState {
	p := new(DetectorState)
	*p = x
	return p
}

func (x DetectorState) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DetectorState) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_dcs_proto_enumTypes[2].Descriptor()
}

func (DetectorState) Type() protoreflect.EnumType {
	return &file_protos_dcs_proto_enumTypes[2]
}

func (x DetectorState) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DetectorState.Descriptor instead.
func (DetectorState) EnumDescriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{2}
}

type RunType int32

const (
	RunType_RT_NULL      RunType = 0
	RunType_RT_PHYSICS   RunType = 1
	RunType_RT_TECHNICAL RunType = 2
)

// Enum value maps for RunType.
var (
	RunType_name = map[int32]string{
		0: "RT_NULL",
		1: "RT_PHYSICS",
		2: "RT_TECHNICAL",
	}
	RunType_value = map[string]int32{
		"RT_NULL":      0,
		"RT_PHYSICS":   1,
		"RT_TECHNICAL": 2,
	}
)

func (x RunType) Enum() *RunType {
	p := new(RunType)
	*p = x
	return p
}

func (x RunType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RunType) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_dcs_proto_enumTypes[3].Descriptor()
}

func (RunType) Type() protoreflect.EnumType {
	return &file_protos_dcs_proto_enumTypes[3]
}

func (x RunType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RunType.Descriptor instead.
func (RunType) EnumDescriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{3}
}

type TriggerMode int32

const (
	TriggerMode_TM_CONTINUOUS TriggerMode = 0
	TriggerMode_TM_TRIGGERED  TriggerMode = 1
)

// Enum value maps for TriggerMode.
var (
	TriggerMode_name = map[int32]string{
		0: "TM_CONTINUOUS",
		1: "TM_TRIGGERED",
	}
	TriggerMode_value = map[string]int32{
		"TM_CONTINUOUS": 0,
		"TM_TRIGGERED":  1,
	}
)

func (x TriggerMode) Enum() *TriggerMode {
	p := new(TriggerMode)
	*p = x
	return p
}

func (x TriggerMode) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (TriggerMode) Descriptor() protoreflect.EnumDescriptor {
	return file_protos_dcs_proto_enumTypes[4].Descriptor()
}

func (TriggerMode) Type() protoreflect.EnumType {
	return &file_protos_dcs_proto_enumTypes[4]
}

func (x TriggerMode) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use TriggerMode.Descriptor instead.
func (TriggerMode) EnumDescriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{4}
}

type SubscriptionRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	InstanceId string `protobuf:"bytes,1,opt,name=instanceId,proto3" json:"instanceId,omitempty"`
}

func (x *SubscriptionRequest) Reset() {
	*x = SubscriptionRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SubscriptionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SubscriptionRequest) ProtoMessage() {}

func (x *SubscriptionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SubscriptionRequest.ProtoReflect.Descriptor instead.
func (*SubscriptionRequest) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{0}
}

func (x *SubscriptionRequest) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

type Event struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Eventtype  EventType `protobuf:"varint,1,opt,name=eventtype,proto3,enum=dcs.EventType" json:"eventtype,omitempty"`
	Detector   Detector  `protobuf:"varint,2,opt,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"`
	Parameters string    `protobuf:"bytes,3,opt,name=parameters,proto3" json:"parameters,omitempty"`
	Timestamp  string    `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *Event) Reset() {
	*x = Event{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Event) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Event) ProtoMessage() {}

func (x *Event) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Event.ProtoReflect.Descriptor instead.
func (*Event) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{1}
}

func (x *Event) GetEventtype() EventType {
	if x != nil {
		return x.Eventtype
	}
	return EventType_NULL_EVENT
}

func (x *Event) GetDetector() Detector {
	if x != nil {
		return x.Detector
	}
	return Detector_NULL_DETECTOR
}

func (x *Event) GetParameters() string {
	if x != nil {
		return x.Parameters
	}
	return ""
}

func (x *Event) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

type SorRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Detector  []Detector `protobuf:"varint,1,rep,packed,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"` // or repeated Detector detectors = 1; if we want to allow multiple detectors per SOR command
	RunType   RunType    `protobuf:"varint,2,opt,name=runType,proto3,enum=dcs.RunType" json:"runType,omitempty"`
	RunNumber int32      `protobuf:"varint,3,opt,name=runNumber,proto3" json:"runNumber,omitempty"`
	//TriggerMode triggerMode = 4 // this is missing, but can be in the parameters
	Parameters map[string]string `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // this can be a map or something strongly typed as we figure it out
}

func (x *SorRequest) Reset() {
	*x = SorRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SorRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SorRequest) ProtoMessage() {}

func (x *SorRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SorRequest.ProtoReflect.Descriptor instead.
func (*SorRequest) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{2}
}

func (x *SorRequest) GetDetector() []Detector {
	if x != nil {
		return x.Detector
	}
	return nil
}

func (x *SorRequest) GetRunType() RunType {
	if x != nil {
		return x.RunType
	}
	return RunType_RT_NULL
}

func (x *SorRequest) GetRunNumber() int32 {
	if x != nil {
		return x.RunNumber
	}
	return 0
}

func (x *SorRequest) GetParameters() map[string]string {
	if x != nil {
		return x.Parameters
	}
	return nil
}

type EorRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Detector   []Detector        `protobuf:"varint,1,rep,packed,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"` // or repeated Detector detectors = 1; if we want to allow multiple detectors per EOR command
	RunNumber  int32             `protobuf:"varint,2,opt,name=runNumber,proto3" json:"runNumber,omitempty"`
	Parameters map[string]string `protobuf:"bytes,3,rep,name=parameters,proto3" json:"parameters,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // does EOR need other params?
}

func (x *EorRequest) Reset() {
	*x = EorRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EorRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EorRequest) ProtoMessage() {}

func (x *EorRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EorRequest.ProtoReflect.Descriptor instead.
func (*EorRequest) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{3}
}

func (x *EorRequest) GetDetector() []Detector {
	if x != nil {
		return x.Detector
	}
	return nil
}

func (x *EorRequest) GetRunNumber() int32 {
	if x != nil {
		return x.RunNumber
	}
	return 0
}

func (x *EorRequest) GetParameters() map[string]string {
	if x != nil {
		return x.Parameters
	}
	return nil
}

type StatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Detector []Detector `protobuf:"varint,1,rep,packed,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"`
}

func (x *StatusRequest) Reset() {
	*x = StatusRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusRequest) ProtoMessage() {}

func (x *StatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusRequest.ProtoReflect.Descriptor instead.
func (*StatusRequest) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{4}
}

func (x *StatusRequest) GetDetector() []Detector {
	if x != nil {
		return x.Detector
	}
	return nil
}

type DetectorInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Detector  Detector      `protobuf:"varint,1,opt,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"`
	State     DetectorState `protobuf:"varint,2,opt,name=state,proto3,enum=dcs.DetectorState" json:"state,omitempty"`
	Timestamp string        `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"` //repeated RunType allowedRunTypes = 3;  //?what is this
}

func (x *DetectorInfo) Reset() {
	*x = DetectorInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DetectorInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DetectorInfo) ProtoMessage() {}

func (x *DetectorInfo) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DetectorInfo.ProtoReflect.Descriptor instead.
func (*DetectorInfo) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{5}
}

func (x *DetectorInfo) GetDetector() Detector {
	if x != nil {
		return x.Detector
	}
	return Detector_NULL_DETECTOR
}

func (x *DetectorInfo) GetState() DetectorState {
	if x != nil {
		return x.State
	}
	return DetectorState_NULL_STATE
}

func (x *DetectorInfo) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

type StatusReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	//repeated DetectorInfo detectorMatrix = 1;
	Detector  Detector      `protobuf:"varint,1,opt,name=detector,proto3,enum=dcs.Detector" json:"detector,omitempty"`
	State     DetectorState `protobuf:"varint,2,opt,name=state,proto3,enum=dcs.DetectorState" json:"state,omitempty"`
	Timestamp string        `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"` //repeated RunType allowedRunTypes = 3;  //?
}

func (x *StatusReply) Reset() {
	*x = StatusReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protos_dcs_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusReply) ProtoMessage() {}

func (x *StatusReply) ProtoReflect() protoreflect.Message {
	mi := &file_protos_dcs_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusReply.ProtoReflect.Descriptor instead.
func (*StatusReply) Descriptor() ([]byte, []int) {
	return file_protos_dcs_proto_rawDescGZIP(), []int{6}
}

func (x *StatusReply) GetDetector() Detector {
	if x != nil {
		return x.Detector
	}
	return Detector_NULL_DETECTOR
}

func (x *StatusReply) GetState() DetectorState {
	if x != nil {
		return x.State
	}
	return DetectorState_NULL_STATE
}

func (x *StatusReply) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

var File_protos_dcs_proto protoreflect.FileDescriptor

var file_protos_dcs_proto_rawDesc = []byte{
	0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x64, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x03, 0x64, 0x63, 0x73, 0x22, 0x35, 0x0a, 0x13, 0x53, 0x75, 0x62, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1e,
	0x0a, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x64, 0x22, 0x9e,
	0x01, 0x0a, 0x05, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x2c, 0x0a, 0x09, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x64, 0x63,
	0x73, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x52, 0x09, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x74, 0x79, 0x70, 0x65, 0x12, 0x29, 0x0a, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44,
	0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72,
	0x73, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22,
	0xfd, 0x01, 0x0a, 0x0a, 0x53, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29,
	0x0a, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0e,
	0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52,
	0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x26, 0x0a, 0x07, 0x72, 0x75, 0x6e,
	0x54, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0c, 0x2e, 0x64, 0x63, 0x73,
	0x2e, 0x52, 0x75, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x07, 0x72, 0x75, 0x6e, 0x54, 0x79, 0x70,
	0x65, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x75, 0x6e, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x72, 0x75, 0x6e, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12,
	0x3f, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x53, 0x6f, 0x72, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73,
	0x1a, 0x3d, 0x0a, 0x0f, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0xd5, 0x01, 0x0a, 0x0a, 0x45, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29,
	0x0a, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0e,
	0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52,
	0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x75, 0x6e,
	0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x72, 0x75,
	0x6e, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x3f, 0x0a, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d,
	0x65, 0x74, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x64, 0x63,
	0x73, 0x2e, 0x45, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x50, 0x61, 0x72,
	0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x70, 0x61,
	0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x1a, 0x3d, 0x0a, 0x0f, 0x50, 0x61, 0x72, 0x61,
	0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x3a, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29, 0x0a, 0x08, 0x64, 0x65, 0x74, 0x65,
	0x63, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0e, 0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73,
	0x2e, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x22, 0x81, 0x01, 0x0a, 0x0c, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x49, 0x6e, 0x66, 0x6f, 0x12, 0x29, 0x0a, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44, 0x65, 0x74,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12,
	0x28, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12,
	0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x80, 0x01, 0x0a, 0x0b, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x29, 0x0a, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0d, 0x2e, 0x64, 0x63, 0x73, 0x2e,
	0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x08, 0x64, 0x65, 0x74, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x12, 0x28, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x12, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x1c, 0x0a, 0x09,
	0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2a, 0x80, 0x01, 0x0a, 0x09, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0e, 0x0a, 0x0a, 0x4e, 0x55, 0x4c, 0x4c,
	0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x48, 0x45, 0x41, 0x52,
	0x54, 0x42, 0x45, 0x41, 0x54, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x54, 0x41, 0x54, 0x45,
	0x5f, 0x43, 0x48, 0x41, 0x4e, 0x47, 0x45, 0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x02, 0x12,
	0x0f, 0x0a, 0x0b, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x03,
	0x12, 0x0d, 0x0a, 0x09, 0x41, 0x43, 0x4b, 0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x04, 0x12,
	0x0d, 0x0a, 0x09, 0x53, 0x4f, 0x52, 0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x14, 0x12, 0x0d,
	0x0a, 0x09, 0x45, 0x4f, 0x52, 0x5f, 0x45, 0x56, 0x45, 0x4e, 0x54, 0x10, 0x1e, 0x2a, 0xad, 0x01,
	0x0a, 0x08, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x11, 0x0a, 0x0d, 0x4e, 0x55,
	0x4c, 0x4c, 0x5f, 0x44, 0x45, 0x54, 0x45, 0x43, 0x54, 0x4f, 0x52, 0x10, 0x00, 0x12, 0x07, 0x0a,
	0x03, 0x43, 0x50, 0x56, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03, 0x45, 0x4d, 0x43, 0x10, 0x02, 0x12,
	0x07, 0x0a, 0x03, 0x46, 0x44, 0x44, 0x10, 0x03, 0x12, 0x07, 0x0a, 0x03, 0x46, 0x54, 0x30, 0x10,
	0x04, 0x12, 0x07, 0x0a, 0x03, 0x46, 0x56, 0x30, 0x10, 0x05, 0x12, 0x07, 0x0a, 0x03, 0x49, 0x54,
	0x53, 0x10, 0x06, 0x12, 0x07, 0x0a, 0x03, 0x48, 0x4d, 0x50, 0x10, 0x07, 0x12, 0x07, 0x0a, 0x03,
	0x4d, 0x43, 0x48, 0x10, 0x08, 0x12, 0x07, 0x0a, 0x03, 0x4d, 0x46, 0x54, 0x10, 0x09, 0x12, 0x07,
	0x0a, 0x03, 0x4d, 0x49, 0x44, 0x10, 0x0a, 0x12, 0x07, 0x0a, 0x03, 0x50, 0x48, 0x53, 0x10, 0x0b,
	0x12, 0x07, 0x0a, 0x03, 0x54, 0x4f, 0x46, 0x10, 0x0c, 0x12, 0x07, 0x0a, 0x03, 0x54, 0x50, 0x43,
	0x10, 0x0d, 0x12, 0x07, 0x0a, 0x03, 0x54, 0x52, 0x44, 0x10, 0x0e, 0x12, 0x07, 0x0a, 0x03, 0x5a,
	0x44, 0x43, 0x10, 0x0f, 0x12, 0x07, 0x0a, 0x03, 0x44, 0x43, 0x53, 0x10, 0x10, 0x2a, 0x9e, 0x01,
	0x0a, 0x0d, 0x44, 0x65, 0x74, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12,
	0x0e, 0x0a, 0x0a, 0x4e, 0x55, 0x4c, 0x4c, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x45, 0x10, 0x00, 0x12,
	0x09, 0x0a, 0x05, 0x52, 0x45, 0x41, 0x44, 0x59, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x55,
	0x4e, 0x5f, 0x4f, 0x4b, 0x10, 0x02, 0x12, 0x0f, 0x0a, 0x0b, 0x52, 0x55, 0x4e, 0x5f, 0x46, 0x41,
	0x49, 0x4c, 0x55, 0x52, 0x45, 0x10, 0x03, 0x12, 0x13, 0x0a, 0x0f, 0x53, 0x4f, 0x52, 0x5f, 0x50,
	0x52, 0x4f, 0x47, 0x52, 0x45, 0x53, 0x53, 0x49, 0x4e, 0x47, 0x10, 0x04, 0x12, 0x13, 0x0a, 0x0f,
	0x45, 0x4f, 0x52, 0x5f, 0x50, 0x52, 0x4f, 0x47, 0x52, 0x45, 0x53, 0x53, 0x49, 0x4e, 0x47, 0x10,
	0x05, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x4f, 0x52, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52, 0x45,
	0x10, 0x06, 0x12, 0x0f, 0x0a, 0x0b, 0x45, 0x4f, 0x52, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52,
	0x45, 0x10, 0x07, 0x12, 0x09, 0x0a, 0x05, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x08, 0x2a, 0x38,
	0x0a, 0x07, 0x52, 0x75, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x52, 0x54, 0x5f,
	0x4e, 0x55, 0x4c, 0x4c, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x52, 0x54, 0x5f, 0x50, 0x48, 0x59,
	0x53, 0x49, 0x43, 0x53, 0x10, 0x01, 0x12, 0x10, 0x0a, 0x0c, 0x52, 0x54, 0x5f, 0x54, 0x45, 0x43,
	0x48, 0x4e, 0x49, 0x43, 0x41, 0x4c, 0x10, 0x02, 0x2a, 0x32, 0x0a, 0x0b, 0x54, 0x72, 0x69, 0x67,
	0x67, 0x65, 0x72, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x11, 0x0a, 0x0d, 0x54, 0x4d, 0x5f, 0x43, 0x4f,
	0x4e, 0x54, 0x49, 0x4e, 0x55, 0x4f, 0x55, 0x53, 0x10, 0x00, 0x12, 0x10, 0x0a, 0x0c, 0x54, 0x4d,
	0x5f, 0x54, 0x52, 0x49, 0x47, 0x47, 0x45, 0x52, 0x45, 0x44, 0x10, 0x01, 0x32, 0xd8, 0x01, 0x0a,
	0x0c, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x35, 0x0a,
	0x09, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65, 0x12, 0x18, 0x2e, 0x64, 0x63, 0x73,
	0x2e, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x22, 0x00, 0x30, 0x01, 0x12, 0x2d, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x72, 0x74, 0x4f, 0x66, 0x52,
	0x75, 0x6e, 0x12, 0x0f, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x53, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x22,
	0x00, 0x30, 0x01, 0x12, 0x2b, 0x0a, 0x08, 0x45, 0x6e, 0x64, 0x4f, 0x66, 0x52, 0x75, 0x6e, 0x12,
	0x0f, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x45, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x0a, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x22, 0x00, 0x30, 0x01,
	0x12, 0x35, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x2e,
	0x64, 0x63, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x10, 0x2e, 0x64, 0x63, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65,
	0x70, 0x6c, 0x79, 0x22, 0x00, 0x30, 0x01, 0x42, 0x21, 0x0a, 0x11, 0x63, 0x68, 0x2e, 0x63, 0x65,
	0x72, 0x6e, 0x2e, 0x61, 0x6c, 0x69, 0x63, 0x65, 0x2e, 0x64, 0x63, 0x73, 0x5a, 0x0c, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x3b, 0x64, 0x63, 0x73, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_protos_dcs_proto_rawDescOnce sync.Once
	file_protos_dcs_proto_rawDescData = file_protos_dcs_proto_rawDesc
)

func file_protos_dcs_proto_rawDescGZIP() []byte {
	file_protos_dcs_proto_rawDescOnce.Do(func() {
		file_protos_dcs_proto_rawDescData = protoimpl.X.CompressGZIP(file_protos_dcs_proto_rawDescData)
	})
	return file_protos_dcs_proto_rawDescData
}

var file_protos_dcs_proto_enumTypes = make([]protoimpl.EnumInfo, 5)
var file_protos_dcs_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_protos_dcs_proto_goTypes = []interface{}{
	(EventType)(0),              // 0: dcs.EventType
	(Detector)(0),               // 1: dcs.Detector
	(DetectorState)(0),          // 2: dcs.DetectorState
	(RunType)(0),                // 3: dcs.RunType
	(TriggerMode)(0),            // 4: dcs.TriggerMode
	(*SubscriptionRequest)(nil), // 5: dcs.SubscriptionRequest
	(*Event)(nil),               // 6: dcs.Event
	(*SorRequest)(nil),          // 7: dcs.SorRequest
	(*EorRequest)(nil),          // 8: dcs.EorRequest
	(*StatusRequest)(nil),       // 9: dcs.StatusRequest
	(*DetectorInfo)(nil),        // 10: dcs.DetectorInfo
	(*StatusReply)(nil),         // 11: dcs.StatusReply
	nil,                         // 12: dcs.SorRequest.ParametersEntry
	nil,                         // 13: dcs.EorRequest.ParametersEntry
}
var file_protos_dcs_proto_depIdxs = []int32{
	0,  // 0: dcs.Event.eventtype:type_name -> dcs.EventType
	1,  // 1: dcs.Event.detector:type_name -> dcs.Detector
	1,  // 2: dcs.SorRequest.detector:type_name -> dcs.Detector
	3,  // 3: dcs.SorRequest.runType:type_name -> dcs.RunType
	12, // 4: dcs.SorRequest.parameters:type_name -> dcs.SorRequest.ParametersEntry
	1,  // 5: dcs.EorRequest.detector:type_name -> dcs.Detector
	13, // 6: dcs.EorRequest.parameters:type_name -> dcs.EorRequest.ParametersEntry
	1,  // 7: dcs.StatusRequest.detector:type_name -> dcs.Detector
	1,  // 8: dcs.DetectorInfo.detector:type_name -> dcs.Detector
	2,  // 9: dcs.DetectorInfo.state:type_name -> dcs.DetectorState
	1,  // 10: dcs.StatusReply.detector:type_name -> dcs.Detector
	2,  // 11: dcs.StatusReply.state:type_name -> dcs.DetectorState
	5,  // 12: dcs.Configurator.Subscribe:input_type -> dcs.SubscriptionRequest
	7,  // 13: dcs.Configurator.StartOfRun:input_type -> dcs.SorRequest
	8,  // 14: dcs.Configurator.EndOfRun:input_type -> dcs.EorRequest
	9,  // 15: dcs.Configurator.GetStatus:input_type -> dcs.StatusRequest
	6,  // 16: dcs.Configurator.Subscribe:output_type -> dcs.Event
	6,  // 17: dcs.Configurator.StartOfRun:output_type -> dcs.Event
	6,  // 18: dcs.Configurator.EndOfRun:output_type -> dcs.Event
	11, // 19: dcs.Configurator.GetStatus:output_type -> dcs.StatusReply
	16, // [16:20] is the sub-list for method output_type
	12, // [12:16] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_protos_dcs_proto_init() }
func file_protos_dcs_proto_init() {
	if File_protos_dcs_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protos_dcs_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SubscriptionRequest); i {
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
		file_protos_dcs_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Event); i {
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
		file_protos_dcs_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SorRequest); i {
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
		file_protos_dcs_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EorRequest); i {
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
		file_protos_dcs_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusRequest); i {
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
		file_protos_dcs_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DetectorInfo); i {
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
		file_protos_dcs_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusReply); i {
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
			RawDescriptor: file_protos_dcs_proto_rawDesc,
			NumEnums:      5,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_protos_dcs_proto_goTypes,
		DependencyIndexes: file_protos_dcs_proto_depIdxs,
		EnumInfos:         file_protos_dcs_proto_enumTypes,
		MessageInfos:      file_protos_dcs_proto_msgTypes,
	}.Build()
	File_protos_dcs_proto = out.File
	file_protos_dcs_proto_rawDesc = nil
	file_protos_dcs_proto_goTypes = nil
	file_protos_dcs_proto_depIdxs = nil
}
