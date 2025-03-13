# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [apricot.proto](#apricot-proto)
    - [AliasedLinkIDsRequest](#apricot-AliasedLinkIDsRequest)
    - [AliasedLinkIDsResponse](#apricot-AliasedLinkIDsResponse)
    - [CRUCardEndpointResponse](#apricot-CRUCardEndpointResponse)
    - [CRUCardsResponse](#apricot-CRUCardsResponse)
    - [CardRequest](#apricot-CardRequest)
    - [ComponentEntriesQuery](#apricot-ComponentEntriesQuery)
    - [ComponentEntriesResponse](#apricot-ComponentEntriesResponse)
    - [ComponentQuery](#apricot-ComponentQuery)
    - [ComponentRequest](#apricot-ComponentRequest)
    - [ComponentRequest.VarStackEntry](#apricot-ComponentRequest-VarStackEntry)
    - [ComponentResponse](#apricot-ComponentResponse)
    - [ComponentResponseWithLastIndex](#apricot-ComponentResponseWithLastIndex)
    - [DetectorEntriesResponse](#apricot-DetectorEntriesResponse)
    - [DetectorEntriesResponse.DetectorEntriesEntry](#apricot-DetectorEntriesResponse-DetectorEntriesEntry)
    - [DetectorInventoryResponse](#apricot-DetectorInventoryResponse)
    - [DetectorResponse](#apricot-DetectorResponse)
    - [DetectorsRequest](#apricot-DetectorsRequest)
    - [DetectorsResponse](#apricot-DetectorsResponse)
    - [Empty](#apricot-Empty)
    - [GetEntryRequest](#apricot-GetEntryRequest)
    - [GetRuntimeEntriesRequest](#apricot-GetRuntimeEntriesRequest)
    - [GetRuntimeEntryRequest](#apricot-GetRuntimeEntryRequest)
    - [HostEntriesResponse](#apricot-HostEntriesResponse)
    - [HostGetRequest](#apricot-HostGetRequest)
    - [HostRequest](#apricot-HostRequest)
    - [HostsRequest](#apricot-HostsRequest)
    - [ImportComponentConfigurationRequest](#apricot-ImportComponentConfigurationRequest)
    - [ImportComponentConfigurationResponse](#apricot-ImportComponentConfigurationResponse)
    - [LinkIDsRequest](#apricot-LinkIDsRequest)
    - [LinkIDsResponse](#apricot-LinkIDsResponse)
    - [ListComponentEntriesRequest](#apricot-ListComponentEntriesRequest)
    - [ListRuntimeEntriesRequest](#apricot-ListRuntimeEntriesRequest)
    - [RawGetRecursiveRequest](#apricot-RawGetRecursiveRequest)
    - [RunNumberResponse](#apricot-RunNumberResponse)
    - [SetRuntimeEntryRequest](#apricot-SetRuntimeEntryRequest)
    - [StringMap](#apricot-StringMap)
    - [StringMap.StringMapEntry](#apricot-StringMap-StringMapEntry)
  
    - [RunType](#apricot-RunType)
  
    - [Apricot](#apricot-Apricot)
  
- [Scalar Value Types](#scalar-value-types)



<a name="apricot-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## apricot.proto



<a name="apricot-AliasedLinkIDsRequest"></a>

### AliasedLinkIDsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detector | [string](#string) |  |  |
| onlyEnabled | [bool](#bool) |  |  |






<a name="apricot-AliasedLinkIDsResponse"></a>

### AliasedLinkIDsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| aliasedLinkIDs | [string](#string) | repeated |  |






<a name="apricot-CRUCardEndpointResponse"></a>

### CRUCardEndpointResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoints | [string](#string) |  |  |






<a name="apricot-CRUCardsResponse"></a>

### CRUCardsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cards | [string](#string) |  |  |






<a name="apricot-CardRequest"></a>

### CardRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostname | [string](#string) |  |  |
| cardSerial | [string](#string) |  |  |






<a name="apricot-ComponentEntriesQuery"></a>

### ComponentEntriesQuery



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| runType | [RunType](#apricot-RunType) |  |  |
| machineRole | [string](#string) |  |  |






<a name="apricot-ComponentEntriesResponse"></a>

### ComponentEntriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) | repeated |  |






<a name="apricot-ComponentQuery"></a>

### ComponentQuery



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| runType | [RunType](#apricot-RunType) |  |  |
| machineRole | [string](#string) |  |  |
| entry | [string](#string) |  |  |






<a name="apricot-ComponentRequest"></a>

### ComponentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| query | [ComponentQuery](#apricot-ComponentQuery) |  |  |
| processTemplate | [bool](#bool) |  |  |
| varStack | [ComponentRequest.VarStackEntry](#apricot-ComponentRequest-VarStackEntry) | repeated |  |






<a name="apricot-ComponentRequest-VarStackEntry"></a>

### ComponentRequest.VarStackEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="apricot-ComponentResponse"></a>

### ComponentResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) |  |  |






<a name="apricot-ComponentResponseWithLastIndex"></a>

### ComponentResponseWithLastIndex



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) |  |  |
| lastIndex | [uint64](#uint64) |  |  |






<a name="apricot-DetectorEntriesResponse"></a>

### DetectorEntriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detectorEntries | [DetectorEntriesResponse.DetectorEntriesEntry](#apricot-DetectorEntriesResponse-DetectorEntriesEntry) | repeated |  |






<a name="apricot-DetectorEntriesResponse-DetectorEntriesEntry"></a>

### DetectorEntriesResponse.DetectorEntriesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [DetectorInventoryResponse](#apricot-DetectorInventoryResponse) |  |  |






<a name="apricot-DetectorInventoryResponse"></a>

### DetectorInventoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flps | [string](#string) | repeated |  |






<a name="apricot-DetectorResponse"></a>

### DetectorResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [string](#string) |  |  |






<a name="apricot-DetectorsRequest"></a>

### DetectorsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| getAll | [bool](#bool) |  | if false(default) restricts &#34;private&#34; detectors (e.g. TRG) |






<a name="apricot-DetectorsResponse"></a>

### DetectorsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detectors | [string](#string) | repeated |  |






<a name="apricot-Empty"></a>

### Empty







<a name="apricot-GetEntryRequest"></a>

### GetEntryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |






<a name="apricot-GetRuntimeEntriesRequest"></a>

### GetRuntimeEntriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |






<a name="apricot-GetRuntimeEntryRequest"></a>

### GetRuntimeEntryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| key | [string](#string) |  |  |






<a name="apricot-HostEntriesResponse"></a>

### HostEntriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hosts | [string](#string) | repeated |  |






<a name="apricot-HostGetRequest"></a>

### HostGetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detector | [string](#string) |  |  |






<a name="apricot-HostRequest"></a>

### HostRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostname | [string](#string) |  |  |






<a name="apricot-HostsRequest"></a>

### HostsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hosts | [string](#string) | repeated |  |






<a name="apricot-ImportComponentConfigurationRequest"></a>

### ImportComponentConfigurationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query | [ComponentQuery](#apricot-ComponentQuery) |  |  |
| payload | [string](#string) |  |  |
| newComponent | [bool](#bool) |  |  |






<a name="apricot-ImportComponentConfigurationResponse"></a>

### ImportComponentConfigurationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| existingComponentUpdated | [bool](#bool) |  |  |
| existingEntryUpdated | [bool](#bool) |  |  |






<a name="apricot-LinkIDsRequest"></a>

### LinkIDsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostname | [string](#string) |  |  |
| cardSerial | [string](#string) |  |  |
| endpoint | [string](#string) |  |  |
| onlyEnabled | [bool](#bool) |  |  |






<a name="apricot-LinkIDsResponse"></a>

### LinkIDsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| linkIDs | [string](#string) | repeated |  |






<a name="apricot-ListComponentEntriesRequest"></a>

### ListComponentEntriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| query | [ComponentEntriesQuery](#apricot-ComponentEntriesQuery) |  |  |






<a name="apricot-ListRuntimeEntriesRequest"></a>

### ListRuntimeEntriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |






<a name="apricot-RawGetRecursiveRequest"></a>

### RawGetRecursiveRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rawPath | [string](#string) |  |  |






<a name="apricot-RunNumberResponse"></a>

### RunNumberResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| runNumber | [uint32](#uint32) |  |  |






<a name="apricot-SetRuntimeEntryRequest"></a>

### SetRuntimeEntryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| component | [string](#string) |  |  |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="apricot-StringMap"></a>

### StringMap



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stringMap | [StringMap.StringMapEntry](#apricot-StringMap-StringMapEntry) | repeated |  |






<a name="apricot-StringMap-StringMapEntry"></a>

### StringMap.StringMapEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="apricot-RunType"></a>

### RunType


| Name | Number | Description |
| ---- | ------ | ----------- |
| NULL | 0 |  |
| PHYSICS | 1 |  |
| TECHNICAL | 2 |  |
| PEDESTAL | 3 |  |
| PULSER | 4 |  |
| LASER | 5 |  |
| CALIBRATION_ITHR_TUNING | 6 |  |
| CALIBRATION_VCASN_TUNING | 7 |  |
| CALIBRATION_THR_SCAN | 8 |  |
| CALIBRATION_DIGITAL_SCAN | 9 |  |
| CALIBRATION_ANALOG_SCAN | 10 |  |
| CALIBRATION_FHR | 11 |  |
| CALIBRATION_ALPIDE_SCAN | 12 |  |
| CALIBRATION | 13 |  |
| COSMICS | 14 |  |
| SYNTHETIC | 15 |  |
| NOISE | 16 |  |
| CALIBRATION_PULSE_LENGTH | 17 |  |
| CALIBRATION_VRESETD | 18 |  |
| ANY | 300 |  |


 

 


<a name="apricot-Apricot"></a>

### Apricot


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| NewRunNumber | [Empty](#apricot-Empty) | [RunNumberResponse](#apricot-RunNumberResponse) |  |
| GetDefaults | [Empty](#apricot-Empty) | [StringMap](#apricot-StringMap) |  |
| GetVars | [Empty](#apricot-Empty) | [StringMap](#apricot-StringMap) |  |
| RawGetRecursive | [RawGetRecursiveRequest](#apricot-RawGetRecursiveRequest) | [ComponentResponse](#apricot-ComponentResponse) |  |
| ListDetectors | [DetectorsRequest](#apricot-DetectorsRequest) | [DetectorsResponse](#apricot-DetectorsResponse) | Detectors and host inventories |
| GetHostInventory | [HostGetRequest](#apricot-HostGetRequest) | [HostEntriesResponse](#apricot-HostEntriesResponse) |  |
| GetDetectorsInventory | [Empty](#apricot-Empty) | [DetectorEntriesResponse](#apricot-DetectorEntriesResponse) |  |
| GetDetectorForHost | [HostRequest](#apricot-HostRequest) | [DetectorResponse](#apricot-DetectorResponse) |  |
| GetDetectorsForHosts | [HostsRequest](#apricot-HostsRequest) | [DetectorsResponse](#apricot-DetectorsResponse) |  |
| GetCRUCardsForHost | [HostRequest](#apricot-HostRequest) | [CRUCardsResponse](#apricot-CRUCardsResponse) |  |
| GetEndpointsForCRUCard | [CardRequest](#apricot-CardRequest) | [CRUCardEndpointResponse](#apricot-CRUCardEndpointResponse) |  |
| GetLinkIDsForCRUEndpoint | [LinkIDsRequest](#apricot-LinkIDsRequest) | [LinkIDsResponse](#apricot-LinkIDsResponse) |  |
| GetAliasedLinkIDsForDetector | [AliasedLinkIDsRequest](#apricot-AliasedLinkIDsRequest) | [AliasedLinkIDsResponse](#apricot-AliasedLinkIDsResponse) |  |
| GetRuntimeEntry | [GetRuntimeEntryRequest](#apricot-GetRuntimeEntryRequest) | [ComponentResponse](#apricot-ComponentResponse) | Runtime KV calls |
| SetRuntimeEntry | [SetRuntimeEntryRequest](#apricot-SetRuntimeEntryRequest) | [Empty](#apricot-Empty) |  |
| GetRuntimeEntries | [GetRuntimeEntriesRequest](#apricot-GetRuntimeEntriesRequest) | [StringMap](#apricot-StringMap) |  |
| ListRuntimeEntries | [ListRuntimeEntriesRequest](#apricot-ListRuntimeEntriesRequest) | [ComponentEntriesResponse](#apricot-ComponentEntriesResponse) |  |
| ListComponents | [Empty](#apricot-Empty) | [ComponentEntriesResponse](#apricot-ComponentEntriesResponse) | Component configuration calls |
| ListComponentEntries | [ListComponentEntriesRequest](#apricot-ListComponentEntriesRequest) | [ComponentEntriesResponse](#apricot-ComponentEntriesResponse) |  |
| GetComponentConfiguration | [ComponentRequest](#apricot-ComponentRequest) | [ComponentResponse](#apricot-ComponentResponse) |  |
| GetComponentConfigurationWithLastIndex | [ComponentRequest](#apricot-ComponentRequest) | [ComponentResponseWithLastIndex](#apricot-ComponentResponseWithLastIndex) |  |
| ResolveComponentQuery | [ComponentQuery](#apricot-ComponentQuery) | [ComponentQuery](#apricot-ComponentQuery) |  |
| ImportComponentConfiguration | [ImportComponentConfigurationRequest](#apricot-ImportComponentConfigurationRequest) | [ImportComponentConfigurationResponse](#apricot-ImportComponentConfigurationResponse) |  |
| InvalidateComponentTemplateCache | [Empty](#apricot-Empty) | [Empty](#apricot-Empty) |  |

 



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

