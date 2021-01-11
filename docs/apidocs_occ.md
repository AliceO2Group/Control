# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [occ.proto](#occ.proto)
    - [ConfigEntry](#occ_pb.ConfigEntry)
    - [DeviceEvent](#occ_pb.DeviceEvent)
    - [EventStreamReply](#occ_pb.EventStreamReply)
    - [EventStreamRequest](#occ_pb.EventStreamRequest)
    - [GetStateReply](#occ_pb.GetStateReply)
    - [GetStateRequest](#occ_pb.GetStateRequest)
    - [StateStreamReply](#occ_pb.StateStreamReply)
    - [StateStreamRequest](#occ_pb.StateStreamRequest)
    - [TransitionReply](#occ_pb.TransitionReply)
    - [TransitionRequest](#occ_pb.TransitionRequest)
  
    - [DeviceEventType](#occ_pb.DeviceEventType)
    - [StateChangeTrigger](#occ_pb.StateChangeTrigger)
    - [StateType](#occ_pb.StateType)
  
    - [Occ](#occ_pb.Occ)
  
- [Scalar Value Types](#scalar-value-types)



<a name="occ.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## occ.proto



<a name="occ_pb.ConfigEntry"></a>

### ConfigEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="occ_pb.DeviceEvent"></a>

### DeviceEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [DeviceEventType](#occ_pb.DeviceEventType) |  |  |






<a name="occ_pb.EventStreamReply"></a>

### EventStreamReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| event | [DeviceEvent](#occ_pb.DeviceEvent) |  |  |






<a name="occ_pb.EventStreamRequest"></a>

### EventStreamRequest







<a name="occ_pb.GetStateReply"></a>

### GetStateReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [string](#string) |  |  |
| pid | [int32](#int32) |  |  |






<a name="occ_pb.GetStateRequest"></a>

### GetStateRequest







<a name="occ_pb.StateStreamReply"></a>

### StateStreamReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [StateType](#occ_pb.StateType) |  |  |
| state | [string](#string) |  |  |






<a name="occ_pb.StateStreamRequest"></a>

### StateStreamRequest







<a name="occ_pb.TransitionReply"></a>

### TransitionReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trigger | [StateChangeTrigger](#occ_pb.StateChangeTrigger) |  |  |
| state | [string](#string) |  |  |
| transitionEvent | [string](#string) |  |  |
| ok | [bool](#bool) |  |  |






<a name="occ_pb.TransitionRequest"></a>

### TransitionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| srcState | [string](#string) |  |  |
| transitionEvent | [string](#string) |  |  |
| arguments | [ConfigEntry](#occ_pb.ConfigEntry) | repeated |  |





 


<a name="occ_pb.DeviceEventType"></a>

### DeviceEventType


| Name | Number | Description |
| ---- | ------ | ----------- |
| NULL_DEVICE_EVENT | 0 |  |
| END_OF_STREAM | 1 |  |
| BASIC_TASK_TERMINATED | 2 |  |



<a name="occ_pb.StateChangeTrigger"></a>

### StateChangeTrigger


| Name | Number | Description |
| ---- | ------ | ----------- |
| EXECUTOR | 0 |  |
| DEVICE_INTENTIONAL | 1 |  |
| DEVICE_ERROR | 2 |  |



<a name="occ_pb.StateType"></a>

### StateType


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_STABLE | 0 |  |
| STATE_INTERMEDIATE | 1 |  |


 

 


<a name="occ_pb.Occ"></a>

### Occ


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| EventStream | [EventStreamRequest](#occ_pb.EventStreamRequest) | [EventStreamReply](#occ_pb.EventStreamReply) stream | We have to have a notification stream because the FairMQDevice might transition on its own for whatever reason. |
| StateStream | [StateStreamRequest](#occ_pb.StateStreamRequest) | [StateStreamReply](#occ_pb.StateStreamReply) stream |  |
| GetState | [GetStateRequest](#occ_pb.GetStateRequest) | [GetStateReply](#occ_pb.GetStateReply) |  |
| Transition | [TransitionRequest](#occ_pb.TransitionRequest) | [TransitionReply](#occ_pb.TransitionReply) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

