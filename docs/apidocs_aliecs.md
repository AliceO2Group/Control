# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [o2control.proto](#o2control-proto)
    - [AddRepoReply](#o2control-AddRepoReply)
    - [AddRepoRequest](#o2control-AddRepoRequest)
    - [ChannelInfo](#o2control-ChannelInfo)
    - [CleanupTasksReply](#o2control-CleanupTasksReply)
    - [CleanupTasksRequest](#o2control-CleanupTasksRequest)
    - [CommandInfo](#o2control-CommandInfo)
    - [ControlEnvironmentReply](#o2control-ControlEnvironmentReply)
    - [ControlEnvironmentRequest](#o2control-ControlEnvironmentRequest)
    - [DestroyEnvironmentReply](#o2control-DestroyEnvironmentReply)
    - [DestroyEnvironmentRequest](#o2control-DestroyEnvironmentRequest)
    - [Empty](#o2control-Empty)
    - [EnvironmentInfo](#o2control-EnvironmentInfo)
    - [EnvironmentInfo.DefaultsEntry](#o2control-EnvironmentInfo-DefaultsEntry)
    - [EnvironmentInfo.IntegratedServicesDataEntry](#o2control-EnvironmentInfo-IntegratedServicesDataEntry)
    - [EnvironmentInfo.UserVarsEntry](#o2control-EnvironmentInfo-UserVarsEntry)
    - [EnvironmentInfo.VarsEntry](#o2control-EnvironmentInfo-VarsEntry)
    - [EnvironmentOperation](#o2control-EnvironmentOperation)
    - [GetActiveDetectorsReply](#o2control-GetActiveDetectorsReply)
    - [GetAvailableDetectorsReply](#o2control-GetAvailableDetectorsReply)
    - [GetEnvironmentPropertiesReply](#o2control-GetEnvironmentPropertiesReply)
    - [GetEnvironmentPropertiesReply.PropertiesEntry](#o2control-GetEnvironmentPropertiesReply-PropertiesEntry)
    - [GetEnvironmentPropertiesRequest](#o2control-GetEnvironmentPropertiesRequest)
    - [GetEnvironmentReply](#o2control-GetEnvironmentReply)
    - [GetEnvironmentRequest](#o2control-GetEnvironmentRequest)
    - [GetEnvironmentsReply](#o2control-GetEnvironmentsReply)
    - [GetEnvironmentsRequest](#o2control-GetEnvironmentsRequest)
    - [GetFrameworkInfoReply](#o2control-GetFrameworkInfoReply)
    - [GetFrameworkInfoRequest](#o2control-GetFrameworkInfoRequest)
    - [GetRolesReply](#o2control-GetRolesReply)
    - [GetRolesRequest](#o2control-GetRolesRequest)
    - [GetTaskReply](#o2control-GetTaskReply)
    - [GetTaskRequest](#o2control-GetTaskRequest)
    - [GetTasksReply](#o2control-GetTasksReply)
    - [GetTasksRequest](#o2control-GetTasksRequest)
    - [GetWorkflowTemplatesReply](#o2control-GetWorkflowTemplatesReply)
    - [GetWorkflowTemplatesRequest](#o2control-GetWorkflowTemplatesRequest)
    - [IntegratedServiceInfo](#o2control-IntegratedServiceInfo)
    - [ListIntegratedServicesReply](#o2control-ListIntegratedServicesReply)
    - [ListIntegratedServicesReply.ServicesEntry](#o2control-ListIntegratedServicesReply-ServicesEntry)
    - [ListReposReply](#o2control-ListReposReply)
    - [ListReposRequest](#o2control-ListReposRequest)
    - [ModifyEnvironmentReply](#o2control-ModifyEnvironmentReply)
    - [ModifyEnvironmentRequest](#o2control-ModifyEnvironmentRequest)
    - [NewAutoEnvironmentReply](#o2control-NewAutoEnvironmentReply)
    - [NewAutoEnvironmentRequest](#o2control-NewAutoEnvironmentRequest)
    - [NewAutoEnvironmentRequest.VarsEntry](#o2control-NewAutoEnvironmentRequest-VarsEntry)
    - [NewEnvironmentReply](#o2control-NewEnvironmentReply)
    - [NewEnvironmentRequest](#o2control-NewEnvironmentRequest)
    - [NewEnvironmentRequest.VarsEntry](#o2control-NewEnvironmentRequest-VarsEntry)
    - [RefreshReposRequest](#o2control-RefreshReposRequest)
    - [RemoveRepoReply](#o2control-RemoveRepoReply)
    - [RemoveRepoRequest](#o2control-RemoveRepoRequest)
    - [RepoInfo](#o2control-RepoInfo)
    - [RoleInfo](#o2control-RoleInfo)
    - [RoleInfo.ConsolidatedStackEntry](#o2control-RoleInfo-ConsolidatedStackEntry)
    - [RoleInfo.DefaultsEntry](#o2control-RoleInfo-DefaultsEntry)
    - [RoleInfo.UserVarsEntry](#o2control-RoleInfo-UserVarsEntry)
    - [RoleInfo.VarsEntry](#o2control-RoleInfo-VarsEntry)
    - [SetDefaultRepoRequest](#o2control-SetDefaultRepoRequest)
    - [SetEnvironmentPropertiesReply](#o2control-SetEnvironmentPropertiesReply)
    - [SetEnvironmentPropertiesRequest](#o2control-SetEnvironmentPropertiesRequest)
    - [SetEnvironmentPropertiesRequest.PropertiesEntry](#o2control-SetEnvironmentPropertiesRequest-PropertiesEntry)
    - [SetGlobalDefaultRevisionRequest](#o2control-SetGlobalDefaultRevisionRequest)
    - [SetRepoDefaultRevisionReply](#o2control-SetRepoDefaultRevisionReply)
    - [SetRepoDefaultRevisionRequest](#o2control-SetRepoDefaultRevisionRequest)
    - [ShortTaskInfo](#o2control-ShortTaskInfo)
    - [SubscribeRequest](#o2control-SubscribeRequest)
    - [TaskDeploymentInfo](#o2control-TaskDeploymentInfo)
    - [TaskInfo](#o2control-TaskInfo)
    - [TaskInfo.PropertiesEntry](#o2control-TaskInfo-PropertiesEntry)
    - [TeardownReply](#o2control-TeardownReply)
    - [TeardownRequest](#o2control-TeardownRequest)
    - [VarSpecMessage](#o2control-VarSpecMessage)
    - [Version](#o2control-Version)
    - [WorkflowTemplateInfo](#o2control-WorkflowTemplateInfo)
    - [WorkflowTemplateInfo.VarSpecMapEntry](#o2control-WorkflowTemplateInfo-VarSpecMapEntry)
  
    - [ControlEnvironmentRequest.Optype](#o2control-ControlEnvironmentRequest-Optype)
    - [EnvironmentOperation.Optype](#o2control-EnvironmentOperation-Optype)
    - [VarSpecMessage.Type](#o2control-VarSpecMessage-Type)
    - [VarSpecMessage.UiWidget](#o2control-VarSpecMessage-UiWidget)
  
    - [Control](#o2control-Control)
  
- [Scalar Value Types](#scalar-value-types)



<a name="o2control-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## o2control.proto



<a name="o2control-AddRepoReply"></a>

### AddRepoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| newDefaultRevision | [string](#string) |  |  |
| info | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-AddRepoRequest"></a>

### AddRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| defaultRevision | [string](#string) |  |  |






<a name="o2control-ChannelInfo"></a>

### ChannelInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| type | [string](#string) |  |  |
| target | [string](#string) |  |  |






<a name="o2control-CleanupTasksReply"></a>

### CleanupTasksReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| killedTasks | [ShortTaskInfo](#o2control-ShortTaskInfo) | repeated |  |
| runningTasks | [ShortTaskInfo](#o2control-ShortTaskInfo) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-CleanupTasksRequest"></a>

### CleanupTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| taskIds | [string](#string) | repeated |  |






<a name="o2control-CommandInfo"></a>

### CommandInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| env | [string](#string) | repeated |  |
| shell | [bool](#bool) |  |  |
| value | [string](#string) |  |  |
| arguments | [string](#string) | repeated |  |
| user | [string](#string) |  |  |






<a name="o2control-ControlEnvironmentReply"></a>

### ControlEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| state | [string](#string) |  |  |
| currentRunNumber | [uint32](#uint32) |  |  |
| startOfTransition | [int64](#int64) |  | All times are in milliseconds |
| endOfTransition | [int64](#int64) |  |  |
| transitionDuration | [int64](#int64) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-ControlEnvironmentRequest"></a>

### ControlEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [ControlEnvironmentRequest.Optype](#o2control-ControlEnvironmentRequest-Optype) |  |  |
| requestUser | [common.User](#common-User) |  |  |






<a name="o2control-DestroyEnvironmentReply"></a>

### DestroyEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cleanupTasksReply | [CleanupTasksReply](#o2control-CleanupTasksReply) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-DestroyEnvironmentRequest"></a>

### DestroyEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| keepTasks | [bool](#bool) |  |  |
| allowInRunningState | [bool](#bool) |  |  |
| force | [bool](#bool) |  |  |
| requestUser | [common.User](#common-User) |  |  |






<a name="o2control-Empty"></a>

### Empty







<a name="o2control-EnvironmentInfo"></a>

### EnvironmentInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| createdWhen | [int64](#int64) |  | msec |
| state | [string](#string) |  |  |
| tasks | [ShortTaskInfo](#o2control-ShortTaskInfo) | repeated |  |
| rootRole | [string](#string) |  |  |
| currentRunNumber | [uint32](#uint32) |  |  |
| defaults | [EnvironmentInfo.DefaultsEntry](#o2control-EnvironmentInfo-DefaultsEntry) | repeated |  |
| vars | [EnvironmentInfo.VarsEntry](#o2control-EnvironmentInfo-VarsEntry) | repeated |  |
| userVars | [EnvironmentInfo.UserVarsEntry](#o2control-EnvironmentInfo-UserVarsEntry) | repeated |  |
| numberOfFlps | [int32](#int32) |  |  |
| includedDetectors | [string](#string) | repeated |  |
| description | [string](#string) |  |  |
| numberOfHosts | [int32](#int32) |  |  |
| integratedServicesData | [EnvironmentInfo.IntegratedServicesDataEntry](#o2control-EnvironmentInfo-IntegratedServicesDataEntry) | repeated |  |
| numberOfTasks | [int32](#int32) |  |  |
| currentTransition | [string](#string) |  |  |
| numberOfActiveTasks | [int32](#int32) |  |  |
| numberOfInactiveTasks | [int32](#int32) |  |  |






<a name="o2control-EnvironmentInfo-DefaultsEntry"></a>

### EnvironmentInfo.DefaultsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-EnvironmentInfo-IntegratedServicesDataEntry"></a>

### EnvironmentInfo.IntegratedServicesDataEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-EnvironmentInfo-UserVarsEntry"></a>

### EnvironmentInfo.UserVarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-EnvironmentInfo-VarsEntry"></a>

### EnvironmentInfo.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-EnvironmentOperation"></a>

### EnvironmentOperation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [EnvironmentOperation.Optype](#o2control-EnvironmentOperation-Optype) |  |  |
| roleName | [string](#string) |  |  |






<a name="o2control-GetActiveDetectorsReply"></a>

### GetActiveDetectorsReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detectors | [string](#string) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetAvailableDetectorsReply"></a>

### GetAvailableDetectorsReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detectors | [string](#string) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetEnvironmentPropertiesReply"></a>

### GetEnvironmentPropertiesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [GetEnvironmentPropertiesReply.PropertiesEntry](#o2control-GetEnvironmentPropertiesReply-PropertiesEntry) | repeated |  |






<a name="o2control-GetEnvironmentPropertiesReply-PropertiesEntry"></a>

### GetEnvironmentPropertiesReply.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-GetEnvironmentPropertiesRequest"></a>

### GetEnvironmentPropertiesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| queries | [string](#string) | repeated | If len(queries) == 0, we return an empty map. To retrieve all KVs, use query &#39;*&#39; |
| excludeGlobals | [bool](#bool) |  |  |






<a name="o2control-GetEnvironmentReply"></a>

### GetEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [EnvironmentInfo](#o2control-EnvironmentInfo) |  |  |
| workflow | [RoleInfo](#o2control-RoleInfo) |  |  |
| public | [bool](#bool) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetEnvironmentRequest"></a>

### GetEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| showWorkflowTree | [bool](#bool) |  |  |






<a name="o2control-GetEnvironmentsReply"></a>

### GetEnvironmentsReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| frameworkId | [string](#string) |  |  |
| environments | [EnvironmentInfo](#o2control-EnvironmentInfo) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetEnvironmentsRequest"></a>

### GetEnvironmentsRequest
Environment
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| showAll | [bool](#bool) |  |  |
| showTaskInfos | [bool](#bool) |  |  |






<a name="o2control-GetFrameworkInfoReply"></a>

### GetFrameworkInfoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| frameworkId | [string](#string) |  |  |
| environmentsCount | [int32](#int32) |  |  |
| tasksCount | [int32](#int32) |  |  |
| state | [string](#string) |  |  |
| hostsCount | [int32](#int32) |  |  |
| instanceName | [string](#string) |  |  |
| version | [Version](#o2control-Version) |  |  |
| configurationEndpoint | [string](#string) |  |  |
| detectorsInInstance | [string](#string) | repeated |  |
| activeDetectors | [string](#string) | repeated |  |
| availableDetectors | [string](#string) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetFrameworkInfoRequest"></a>

### GetFrameworkInfoRequest
Framework
//////////////////////////////////////






<a name="o2control-GetRolesReply"></a>

### GetRolesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [RoleInfo](#o2control-RoleInfo) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetRolesRequest"></a>

### GetRolesRequest
Roles
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| envId | [string](#string) |  |  |
| pathSpec | [string](#string) |  |  |






<a name="o2control-GetTaskReply"></a>

### GetTaskReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [TaskInfo](#o2control-TaskInfo) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetTaskRequest"></a>

### GetTaskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| taskId | [string](#string) |  |  |






<a name="o2control-GetTasksReply"></a>

### GetTasksReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tasks | [ShortTaskInfo](#o2control-ShortTaskInfo) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetTasksRequest"></a>

### GetTasksRequest







<a name="o2control-GetWorkflowTemplatesReply"></a>

### GetWorkflowTemplatesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplates | [WorkflowTemplateInfo](#o2control-WorkflowTemplateInfo) | repeated |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-GetWorkflowTemplatesRequest"></a>

### GetWorkflowTemplatesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoPattern | [string](#string) |  |  |
| revisionPattern | [string](#string) |  |  |
| allBranches | [bool](#bool) |  |  |
| allTags | [bool](#bool) |  |  |
| allWorkflows | [bool](#bool) |  |  |






<a name="o2control-IntegratedServiceInfo"></a>

### IntegratedServiceInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | user-visible service name, e.g. &#34;DD scheduler&#34; |
| enabled | [bool](#bool) |  |  |
| endpoint | [string](#string) |  |  |
| connectionState | [string](#string) |  | allowed values: READY, CONNECTING, TRANSIENT_FAILURE, IDLE, SHUTDOWN |
| data | [string](#string) |  | always a JSON payload with a map&lt;string, string&gt; inside. |






<a name="o2control-ListIntegratedServicesReply"></a>

### ListIntegratedServicesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| services | [ListIntegratedServicesReply.ServicesEntry](#o2control-ListIntegratedServicesReply-ServicesEntry) | repeated | keys are IDs (e.g. &#34;ddsched&#34;), the service name should be displayed to users instead |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-ListIntegratedServicesReply-ServicesEntry"></a>

### ListIntegratedServicesReply.ServicesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [IntegratedServiceInfo](#o2control-IntegratedServiceInfo) |  |  |






<a name="o2control-ListReposReply"></a>

### ListReposReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repos | [RepoInfo](#o2control-RepoInfo) | repeated |  |
| globalDefaultRevision | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-ListReposRequest"></a>

### ListReposRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| getRevisions | [bool](#bool) |  |  |






<a name="o2control-ModifyEnvironmentReply"></a>

### ModifyEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| failedOperations | [EnvironmentOperation](#o2control-EnvironmentOperation) | repeated |  |
| id | [string](#string) |  |  |
| state | [string](#string) |  |  |






<a name="o2control-ModifyEnvironmentRequest"></a>

### ModifyEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| operations | [EnvironmentOperation](#o2control-EnvironmentOperation) | repeated |  |
| reconfigureAll | [bool](#bool) |  |  |






<a name="o2control-NewAutoEnvironmentReply"></a>

### NewAutoEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-NewAutoEnvironmentRequest"></a>

### NewAutoEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplate | [string](#string) |  |  |
| vars | [NewAutoEnvironmentRequest.VarsEntry](#o2control-NewAutoEnvironmentRequest-VarsEntry) | repeated |  |
| id | [string](#string) |  |  |
| requestUser | [common.User](#common-User) |  |  |






<a name="o2control-NewAutoEnvironmentRequest-VarsEntry"></a>

### NewAutoEnvironmentRequest.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-NewEnvironmentReply"></a>

### NewEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [EnvironmentInfo](#o2control-EnvironmentInfo) |  |  |
| public | [bool](#bool) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-NewEnvironmentRequest"></a>

### NewEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplate | [string](#string) |  |  |
| vars | [NewEnvironmentRequest.VarsEntry](#o2control-NewEnvironmentRequest-VarsEntry) | repeated |  |
| public | [bool](#bool) |  |  |
| autoTransition | [bool](#bool) |  |  |
| requestUser | [common.User](#common-User) |  |  |






<a name="o2control-NewEnvironmentRequest-VarsEntry"></a>

### NewEnvironmentRequest.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-RefreshReposRequest"></a>

### RefreshReposRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control-RemoveRepoReply"></a>

### RemoveRepoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| newDefaultRepo | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-RemoveRepoRequest"></a>

### RemoveRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control-RepoInfo"></a>

### RepoInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| default | [bool](#bool) |  |  |
| defaultRevision | [string](#string) |  |  |
| revisions | [string](#string) | repeated |  |






<a name="o2control-RoleInfo"></a>

### RoleInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| status | [string](#string) |  |  |
| state | [string](#string) |  |  |
| fullPath | [string](#string) |  |  |
| taskIds | [string](#string) | repeated |  |
| roles | [RoleInfo](#o2control-RoleInfo) | repeated |  |
| defaults | [RoleInfo.DefaultsEntry](#o2control-RoleInfo-DefaultsEntry) | repeated |  |
| vars | [RoleInfo.VarsEntry](#o2control-RoleInfo-VarsEntry) | repeated |  |
| userVars | [RoleInfo.UserVarsEntry](#o2control-RoleInfo-UserVarsEntry) | repeated |  |
| consolidatedStack | [RoleInfo.ConsolidatedStackEntry](#o2control-RoleInfo-ConsolidatedStackEntry) | repeated |  |
| description | [string](#string) |  |  |






<a name="o2control-RoleInfo-ConsolidatedStackEntry"></a>

### RoleInfo.ConsolidatedStackEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-RoleInfo-DefaultsEntry"></a>

### RoleInfo.DefaultsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-RoleInfo-UserVarsEntry"></a>

### RoleInfo.UserVarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-RoleInfo-VarsEntry"></a>

### RoleInfo.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-SetDefaultRepoRequest"></a>

### SetDefaultRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control-SetEnvironmentPropertiesReply"></a>

### SetEnvironmentPropertiesReply







<a name="o2control-SetEnvironmentPropertiesRequest"></a>

### SetEnvironmentPropertiesRequest
Environment, GET/SET properties
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| properties | [SetEnvironmentPropertiesRequest.PropertiesEntry](#o2control-SetEnvironmentPropertiesRequest-PropertiesEntry) | repeated | If properties == nil, the core sets nothing and reply ok |






<a name="o2control-SetEnvironmentPropertiesRequest-PropertiesEntry"></a>

### SetEnvironmentPropertiesRequest.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-SetGlobalDefaultRevisionRequest"></a>

### SetGlobalDefaultRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision | [string](#string) |  |  |






<a name="o2control-SetRepoDefaultRevisionReply"></a>

### SetRepoDefaultRevisionReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| info | [string](#string) |  |  |
| timestamp | [int64](#int64) |  | timestamp of when this object was sent in unix milliseconds |






<a name="o2control-SetRepoDefaultRevisionRequest"></a>

### SetRepoDefaultRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |
| revision | [string](#string) |  |  |






<a name="o2control-ShortTaskInfo"></a>

### ShortTaskInfo
Tasks
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| locked | [bool](#bool) |  |  |
| taskId | [string](#string) |  |  |
| status | [string](#string) |  |  |
| state | [string](#string) |  |  |
| className | [string](#string) |  |  |
| deploymentInfo | [TaskDeploymentInfo](#o2control-TaskDeploymentInfo) |  |  |
| pid | [string](#string) |  |  |
| sandboxStdout | [string](#string) |  |  |
| claimable | [bool](#bool) |  |  |
| critical | [bool](#bool) |  |  |






<a name="o2control-SubscribeRequest"></a>

### SubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="o2control-TaskDeploymentInfo"></a>

### TaskDeploymentInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostname | [string](#string) |  |  |
| agentId | [string](#string) |  |  |
| offerId | [string](#string) |  |  |
| executorId | [string](#string) |  |  |






<a name="o2control-TaskInfo"></a>

### TaskInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| shortInfo | [ShortTaskInfo](#o2control-ShortTaskInfo) |  |  |
| inboundChannels | [ChannelInfo](#o2control-ChannelInfo) | repeated |  |
| outboundChannels | [ChannelInfo](#o2control-ChannelInfo) | repeated |  |
| commandInfo | [CommandInfo](#o2control-CommandInfo) |  |  |
| taskPath | [string](#string) |  |  |
| envId | [string](#string) |  |  |
| properties | [TaskInfo.PropertiesEntry](#o2control-TaskInfo-PropertiesEntry) | repeated |  |






<a name="o2control-TaskInfo-PropertiesEntry"></a>

### TaskInfo.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control-TeardownReply"></a>

### TeardownReply







<a name="o2control-TeardownRequest"></a>

### TeardownRequest
Not implemented yet


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reason | [string](#string) |  |  |






<a name="o2control-VarSpecMessage"></a>

### VarSpecMessage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| defaultValue | [string](#string) |  |  |
| type | [VarSpecMessage.Type](#o2control-VarSpecMessage-Type) |  |  |
| label | [string](#string) |  |  |
| description | [string](#string) |  |  |
| widget | [VarSpecMessage.UiWidget](#o2control-VarSpecMessage-UiWidget) |  |  |
| panel | [string](#string) |  | hint for the UI on where to put or group the given variable input |
| allowedValues | [string](#string) | repeated | list of offered values from which to choose (only for some UiWidgets) |
| index | [int32](#int32) |  |  |
| visibleIf | [string](#string) |  | JS expression that evaluates to bool |
| enabledIf | [string](#string) |  | JS expression that evaluates to bool |
| rows | [uint32](#uint32) |  | this field is used only if widget == editBox |






<a name="o2control-Version"></a>

### Version



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| major | [int32](#int32) |  |  |
| minor | [int32](#int32) |  |  |
| patch | [int32](#int32) |  |  |
| build | [string](#string) |  |  |
| productName | [string](#string) |  |  |
| versionStr | [string](#string) |  |  |






<a name="o2control-WorkflowTemplateInfo"></a>

### WorkflowTemplateInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repo | [string](#string) |  |  |
| template | [string](#string) |  |  |
| revision | [string](#string) |  |  |
| varSpecMap | [WorkflowTemplateInfo.VarSpecMapEntry](#o2control-WorkflowTemplateInfo-VarSpecMapEntry) | repeated |  |
| description | [string](#string) |  |  |






<a name="o2control-WorkflowTemplateInfo-VarSpecMapEntry"></a>

### WorkflowTemplateInfo.VarSpecMapEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [VarSpecMessage](#o2control-VarSpecMessage) |  |  |





 


<a name="o2control-ControlEnvironmentRequest-Optype"></a>

### ControlEnvironmentRequest.Optype


| Name | Number | Description |
| ---- | ------ | ----------- |
| NOOP | 0 |  |
| START_ACTIVITY | 1 |  |
| STOP_ACTIVITY | 2 |  |
| CONFIGURE | 3 |  |
| RESET | 4 |  |
| GO_ERROR | 5 |  |
| DEPLOY | 6 |  |



<a name="o2control-EnvironmentOperation-Optype"></a>

### EnvironmentOperation.Optype


| Name | Number | Description |
| ---- | ------ | ----------- |
| NOOP | 0 |  |
| REMOVE_ROLE | 3 |  |
| ADD_ROLE | 4 |  |



<a name="o2control-VarSpecMessage-Type"></a>

### VarSpecMessage.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| string | 0 |  |
| number | 1 |  |
| bool | 2 |  |
| list | 3 |  |
| map | 4 |  |



<a name="o2control-VarSpecMessage-UiWidget"></a>

### VarSpecMessage.UiWidget


| Name | Number | Description |
| ---- | ------ | ----------- |
| editBox | 0 | plain string input line, can accept types number (like a spinBox) and string |
| slider | 1 | input widget exclusively for numbers, range allowedValues[0]-[1] |
| listBox | 2 | displays a list of items, can accept types number, string or list; if number/string ==&gt; single selection, otherwise multiple selection allowed |
| dropDownBox | 3 |  |
| comboBox | 4 |  |
| radioButtonBox | 5 |  |
| checkBox | 6 |  |


 

 


<a name="o2control-Control"></a>

### Control


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetFrameworkInfo | [GetFrameworkInfoRequest](#o2control-GetFrameworkInfoRequest) | [GetFrameworkInfoReply](#o2control-GetFrameworkInfoReply) |  |
| GetEnvironments | [GetEnvironmentsRequest](#o2control-GetEnvironmentsRequest) | [GetEnvironmentsReply](#o2control-GetEnvironmentsReply) |  |
| NewAutoEnvironment | [NewAutoEnvironmentRequest](#o2control-NewAutoEnvironmentRequest) | [NewAutoEnvironmentReply](#o2control-NewAutoEnvironmentReply) |  |
| NewEnvironment | [NewEnvironmentRequest](#o2control-NewEnvironmentRequest) | [NewEnvironmentReply](#o2control-NewEnvironmentReply) |  |
| GetEnvironment | [GetEnvironmentRequest](#o2control-GetEnvironmentRequest) | [GetEnvironmentReply](#o2control-GetEnvironmentReply) |  |
| ControlEnvironment | [ControlEnvironmentRequest](#o2control-ControlEnvironmentRequest) | [ControlEnvironmentReply](#o2control-ControlEnvironmentReply) |  |
| DestroyEnvironment | [DestroyEnvironmentRequest](#o2control-DestroyEnvironmentRequest) | [DestroyEnvironmentReply](#o2control-DestroyEnvironmentReply) |  |
| GetActiveDetectors | [Empty](#o2control-Empty) | [GetActiveDetectorsReply](#o2control-GetActiveDetectorsReply) |  |
| GetAvailableDetectors | [Empty](#o2control-Empty) | [GetAvailableDetectorsReply](#o2control-GetAvailableDetectorsReply) |  |
| NewEnvironmentAsync | [NewEnvironmentRequest](#o2control-NewEnvironmentRequest) | [NewEnvironmentReply](#o2control-NewEnvironmentReply) |  |
| GetTasks | [GetTasksRequest](#o2control-GetTasksRequest) | [GetTasksReply](#o2control-GetTasksReply) |  |
| GetTask | [GetTaskRequest](#o2control-GetTaskRequest) | [GetTaskReply](#o2control-GetTaskReply) |  |
| CleanupTasks | [CleanupTasksRequest](#o2control-CleanupTasksRequest) | [CleanupTasksReply](#o2control-CleanupTasksReply) |  |
| GetRoles | [GetRolesRequest](#o2control-GetRolesRequest) | [GetRolesReply](#o2control-GetRolesReply) |  |
| GetWorkflowTemplates | [GetWorkflowTemplatesRequest](#o2control-GetWorkflowTemplatesRequest) | [GetWorkflowTemplatesReply](#o2control-GetWorkflowTemplatesReply) |  |
| ListRepos | [ListReposRequest](#o2control-ListReposRequest) | [ListReposReply](#o2control-ListReposReply) |  |
| AddRepo | [AddRepoRequest](#o2control-AddRepoRequest) | [AddRepoReply](#o2control-AddRepoReply) |  |
| RemoveRepo | [RemoveRepoRequest](#o2control-RemoveRepoRequest) | [RemoveRepoReply](#o2control-RemoveRepoReply) |  |
| RefreshRepos | [RefreshReposRequest](#o2control-RefreshReposRequest) | [Empty](#o2control-Empty) |  |
| SetDefaultRepo | [SetDefaultRepoRequest](#o2control-SetDefaultRepoRequest) | [Empty](#o2control-Empty) |  |
| SetGlobalDefaultRevision | [SetGlobalDefaultRevisionRequest](#o2control-SetGlobalDefaultRevisionRequest) | [Empty](#o2control-Empty) |  |
| SetRepoDefaultRevision | [SetRepoDefaultRevisionRequest](#o2control-SetRepoDefaultRevisionRequest) | [SetRepoDefaultRevisionReply](#o2control-SetRepoDefaultRevisionReply) |  |
| Subscribe | [SubscribeRequest](#o2control-SubscribeRequest) | [.events.Event](#events-Event) stream |  |
| GetIntegratedServices | [Empty](#o2control-Empty) | [ListIntegratedServicesReply](#o2control-ListIntegratedServicesReply) |  |
| Teardown | [TeardownRequest](#o2control-TeardownRequest) | [TeardownReply](#o2control-TeardownReply) | Reserved and not implemented: |
| ModifyEnvironment | [ModifyEnvironmentRequest](#o2control-ModifyEnvironmentRequest) | [ModifyEnvironmentReply](#o2control-ModifyEnvironmentReply) |  |

 



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

