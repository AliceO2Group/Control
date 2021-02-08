# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [apricot.proto](#apricot.proto)
    - [ComponentEntriesQuery](#apricot.ComponentEntriesQuery)
    - [ComponentEntriesResponse](#apricot.ComponentEntriesResponse)
    - [ComponentQuery](#apricot.ComponentQuery)
    - [ComponentRequest](#apricot.ComponentRequest)
    - [ComponentRequest.VarStackEntry](#apricot.ComponentRequest.VarStackEntry)
    - [ComponentResponse](#apricot.ComponentResponse)
    - [Empty](#apricot.Empty)
    - [GetRuntimeEntryRequest](#apricot.GetRuntimeEntryRequest)
    - [ImportComponentConfigurationRequest](#apricot.ImportComponentConfigurationRequest)
    - [ImportComponentConfigurationResponse](#apricot.ImportComponentConfigurationResponse)
    - [ListComponentEntriesRequest](#apricot.ListComponentEntriesRequest)
    - [RawGetRecursiveRequest](#apricot.RawGetRecursiveRequest)
    - [RunNumberResponse](#apricot.RunNumberResponse)
    - [SetRuntimeEntryRequest](#apricot.SetRuntimeEntryRequest)
    - [StringMap](#apricot.StringMap)
    - [StringMap.StringMapEntry](#apricot.StringMap.StringMapEntry)
  
    - [RunType](#apricot.RunType)
  
    - [Apricot](#apricot.Apricot)
  
- [Scalar Value Types](#scalar-value-types)



<a name="apricot.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## apricot.proto



<a name="apricot.ComponentEntriesQuery"></a>

### ComponentEntriesQuery



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| runType | [RunType](#apricot.RunType) |  |  |
| machineRole | [string](#string) |  |  |






<a name="apricot.ComponentEntriesResponse"></a>

### ComponentEntriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) | repeated |  |






<a name="apricot.ComponentQuery"></a>

### ComponentQuery



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| runType | [RunType](#apricot.RunType) |  |  |
| machineRole | [string](#string) |  |  |
| entry | [string](#string) |  |  |
| timestamp | [string](#string) |  |  |






<a name="apricot.ComponentRequest"></a>

### ComponentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| query | [ComponentQuery](#apricot.ComponentQuery) |  |  |
| processTemplate | [bool](#bool) |  |  |
| varStack | [ComponentRequest.VarStackEntry](#apricot.ComponentRequest.VarStackEntry) | repeated |  |






<a name="apricot.ComponentRequest.VarStackEntry"></a>

### ComponentRequest.VarStackEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="apricot.ComponentResponse"></a>

### ComponentResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) |  |  |






<a name="apricot.Empty"></a>

### Empty







<a name="apricot.GetRuntimeEntryRequest"></a>

### GetRuntimeEntryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| key | [string](#string) |  |  |






<a name="apricot.ImportComponentConfigurationRequest"></a>

### ImportComponentConfigurationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query | [ComponentQuery](#apricot.ComponentQuery) |  |  |
| payload | [string](#string) |  |  |
| newComponent | [bool](#bool) |  |  |
| useVersioning | [bool](#bool) |  |  |






<a name="apricot.ImportComponentConfigurationResponse"></a>

### ImportComponentConfigurationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| existingComponentUpdated | [bool](#bool) |  |  |
| existingEntryUpdated | [bool](#bool) |  |  |
| newTimestamp | [int64](#int64) |  |  |






<a name="apricot.ListComponentEntriesRequest"></a>

### ListComponentEntriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| query | [ComponentEntriesQuery](#apricot.ComponentEntriesQuery) |  |  |
| includeTimestamps | [bool](#bool) |  |  |






<a name="apricot.RawGetRecursiveRequest"></a>

### RawGetRecursiveRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rawPath | [string](#string) |  |  |






<a name="apricot.RunNumberResponse"></a>

### RunNumberResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| runNumber | [uint32](#uint32) |  |  |






<a name="apricot.SetRuntimeEntryRequest"></a>

### SetRuntimeEntryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="apricot.StringMap"></a>

### StringMap



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stringMap | [StringMap.StringMapEntry](#apricot.StringMap.StringMapEntry) | repeated |  |






<a name="apricot.StringMap.StringMapEntry"></a>

### StringMap.StringMapEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="apricot.RunType"></a>

### RunType


| Name | Number | Description |
| ---- | ------ | ----------- |
| NULL | 0 |  |
| ANY | 1 |  |
| PHYSICS | 2 |  |
| TECHNICAL | 3 |  |
| COSMIC | 4 |  |
| PEDESTAL | 5 |  |
| THRESHOLD_SCAN | 6 |  |
| CALIBRATION | 7 |  |


 

 


<a name="apricot.Apricot"></a>

### Apricot


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| NewRunNumber | [Empty](#apricot.Empty) | [RunNumberResponse](#apricot.RunNumberResponse) |  |
| GetDefaults | [Empty](#apricot.Empty) | [StringMap](#apricot.StringMap) |  |
| GetVars | [Empty](#apricot.Empty) | [StringMap](#apricot.StringMap) |  |
| RawGetRecursive | [RawGetRecursiveRequest](#apricot.RawGetRecursiveRequest) | [ComponentResponse](#apricot.ComponentResponse) |  |
| GetRuntimeEntry | [GetRuntimeEntryRequest](#apricot.GetRuntimeEntryRequest) | [ComponentResponse](#apricot.ComponentResponse) |  |
| SetRuntimeEntry | [SetRuntimeEntryRequest](#apricot.SetRuntimeEntryRequest) | [Empty](#apricot.Empty) |  |
| ListComponents | [Empty](#apricot.Empty) | [ComponentEntriesResponse](#apricot.ComponentEntriesResponse) |  |
| ListComponentEntries | [ListComponentEntriesRequest](#apricot.ListComponentEntriesRequest) | [ComponentEntriesResponse](#apricot.ComponentEntriesResponse) |  |
| ListComponentEntryHistory | [ComponentQuery](#apricot.ComponentQuery) | [ComponentEntriesResponse](#apricot.ComponentEntriesResponse) |  |
| GetComponentConfiguration | [ComponentRequest](#apricot.ComponentRequest) | [ComponentResponse](#apricot.ComponentResponse) |  |
| ImportComponentConfiguration | [ImportComponentConfigurationRequest](#apricot.ImportComponentConfigurationRequest) | [ImportComponentConfigurationResponse](#apricot.ImportComponentConfigurationResponse) |  |

 



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

