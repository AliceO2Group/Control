# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [o2control.proto](#o2control.proto)
    - [AddRepoReply](#o2control.AddRepoReply)
    - [AddRepoRequest](#o2control.AddRepoRequest)
    - [ChannelInfo](#o2control.ChannelInfo)
    - [CleanupTasksReply](#o2control.CleanupTasksReply)
    - [CleanupTasksRequest](#o2control.CleanupTasksRequest)
    - [CommandInfo](#o2control.CommandInfo)
    - [ControlEnvironmentReply](#o2control.ControlEnvironmentReply)
    - [ControlEnvironmentRequest](#o2control.ControlEnvironmentRequest)
    - [DestroyEnvironmentReply](#o2control.DestroyEnvironmentReply)
    - [DestroyEnvironmentRequest](#o2control.DestroyEnvironmentRequest)
    - [Empty](#o2control.Empty)
    - [EnvironmentInfo](#o2control.EnvironmentInfo)
    - [EnvironmentInfo.DefaultsEntry](#o2control.EnvironmentInfo.DefaultsEntry)
    - [EnvironmentInfo.UserVarsEntry](#o2control.EnvironmentInfo.UserVarsEntry)
    - [EnvironmentInfo.VarsEntry](#o2control.EnvironmentInfo.VarsEntry)
    - [EnvironmentOperation](#o2control.EnvironmentOperation)
    - [Ev_EnvironmentEvent](#o2control.Ev_EnvironmentEvent)
    - [Ev_RoleEvent](#o2control.Ev_RoleEvent)
    - [Ev_TaskEvent](#o2control.Ev_TaskEvent)
    - [Event](#o2control.Event)
    - [Event_MesosHeartbeat](#o2control.Event_MesosHeartbeat)
    - [GetEnvironmentPropertiesReply](#o2control.GetEnvironmentPropertiesReply)
    - [GetEnvironmentPropertiesReply.PropertiesEntry](#o2control.GetEnvironmentPropertiesReply.PropertiesEntry)
    - [GetEnvironmentPropertiesRequest](#o2control.GetEnvironmentPropertiesRequest)
    - [GetEnvironmentReply](#o2control.GetEnvironmentReply)
    - [GetEnvironmentRequest](#o2control.GetEnvironmentRequest)
    - [GetEnvironmentsReply](#o2control.GetEnvironmentsReply)
    - [GetEnvironmentsRequest](#o2control.GetEnvironmentsRequest)
    - [GetFrameworkInfoReply](#o2control.GetFrameworkInfoReply)
    - [GetFrameworkInfoRequest](#o2control.GetFrameworkInfoRequest)
    - [GetRolesReply](#o2control.GetRolesReply)
    - [GetRolesRequest](#o2control.GetRolesRequest)
    - [GetTaskReply](#o2control.GetTaskReply)
    - [GetTaskRequest](#o2control.GetTaskRequest)
    - [GetTasksReply](#o2control.GetTasksReply)
    - [GetTasksRequest](#o2control.GetTasksRequest)
    - [GetWorkflowTemplatesReply](#o2control.GetWorkflowTemplatesReply)
    - [GetWorkflowTemplatesRequest](#o2control.GetWorkflowTemplatesRequest)
    - [ListReposReply](#o2control.ListReposReply)
    - [ListReposRequest](#o2control.ListReposRequest)
    - [ModifyEnvironmentReply](#o2control.ModifyEnvironmentReply)
    - [ModifyEnvironmentRequest](#o2control.ModifyEnvironmentRequest)
    - [NewAutoEnvironmentReply](#o2control.NewAutoEnvironmentReply)
    - [NewAutoEnvironmentRequest](#o2control.NewAutoEnvironmentRequest)
    - [NewAutoEnvironmentRequest.VarsEntry](#o2control.NewAutoEnvironmentRequest.VarsEntry)
    - [NewEnvironmentReply](#o2control.NewEnvironmentReply)
    - [NewEnvironmentRequest](#o2control.NewEnvironmentRequest)
    - [NewEnvironmentRequest.VarsEntry](#o2control.NewEnvironmentRequest.VarsEntry)
    - [RefreshReposRequest](#o2control.RefreshReposRequest)
    - [RemoveRepoReply](#o2control.RemoveRepoReply)
    - [RemoveRepoRequest](#o2control.RemoveRepoRequest)
    - [RepoInfo](#o2control.RepoInfo)
    - [RoleInfo](#o2control.RoleInfo)
    - [RoleInfo.DefaultsEntry](#o2control.RoleInfo.DefaultsEntry)
    - [RoleInfo.UserVarsEntry](#o2control.RoleInfo.UserVarsEntry)
    - [RoleInfo.VarsEntry](#o2control.RoleInfo.VarsEntry)
    - [SetDefaultRepoRequest](#o2control.SetDefaultRepoRequest)
    - [SetEnvironmentPropertiesReply](#o2control.SetEnvironmentPropertiesReply)
    - [SetEnvironmentPropertiesRequest](#o2control.SetEnvironmentPropertiesRequest)
    - [SetEnvironmentPropertiesRequest.PropertiesEntry](#o2control.SetEnvironmentPropertiesRequest.PropertiesEntry)
    - [SetGlobalDefaultRevisionRequest](#o2control.SetGlobalDefaultRevisionRequest)
    - [SetRepoDefaultRevisionReply](#o2control.SetRepoDefaultRevisionReply)
    - [SetRepoDefaultRevisionRequest](#o2control.SetRepoDefaultRevisionRequest)
    - [ShortTaskInfo](#o2control.ShortTaskInfo)
    - [StatusReply](#o2control.StatusReply)
    - [StatusRequest](#o2control.StatusRequest)
    - [StatusUpdate](#o2control.StatusUpdate)
    - [SubscribeRequest](#o2control.SubscribeRequest)
    - [TaskClassInfo](#o2control.TaskClassInfo)
    - [TaskDeploymentInfo](#o2control.TaskDeploymentInfo)
    - [TaskInfo](#o2control.TaskInfo)
    - [TaskInfo.PropertiesEntry](#o2control.TaskInfo.PropertiesEntry)
    - [TeardownReply](#o2control.TeardownReply)
    - [TeardownRequest](#o2control.TeardownRequest)
    - [Version](#o2control.Version)
    - [WorkflowTemplateInfo](#o2control.WorkflowTemplateInfo)
  
    - [ControlEnvironmentRequest.Optype](#o2control.ControlEnvironmentRequest.Optype)
    - [EnvironmentOperation.Optype](#o2control.EnvironmentOperation.Optype)
    - [StatusUpdate.Level](#o2control.StatusUpdate.Level)
  
    - [Control](#o2control.Control)
  
- [Scalar Value Types](#scalar-value-types)



<a name="o2control.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## o2control.proto



<a name="o2control.AddRepoReply"></a>

### AddRepoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| newDefaultRevision | [string](#string) |  |  |
| info | [string](#string) |  |  |






<a name="o2control.AddRepoRequest"></a>

### AddRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| defaultRevision | [string](#string) |  |  |






<a name="o2control.ChannelInfo"></a>

### ChannelInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| type | [string](#string) |  |  |
| target | [string](#string) |  |  |






<a name="o2control.CleanupTasksReply"></a>

### CleanupTasksReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| killedTasks | [ShortTaskInfo](#o2control.ShortTaskInfo) | repeated |  |
| runningTasks | [ShortTaskInfo](#o2control.ShortTaskInfo) | repeated |  |






<a name="o2control.CleanupTasksRequest"></a>

### CleanupTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| taskIds | [string](#string) | repeated |  |






<a name="o2control.CommandInfo"></a>

### CommandInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| env | [string](#string) | repeated |  |
| shell | [bool](#bool) |  |  |
| value | [string](#string) |  |  |
| arguments | [string](#string) | repeated |  |
| user | [string](#string) |  |  |






<a name="o2control.ControlEnvironmentReply"></a>

### ControlEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| state | [string](#string) |  |  |
| currentRunNumber | [uint32](#uint32) |  |  |






<a name="o2control.ControlEnvironmentRequest"></a>

### ControlEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [ControlEnvironmentRequest.Optype](#o2control.ControlEnvironmentRequest.Optype) |  |  |






<a name="o2control.DestroyEnvironmentReply"></a>

### DestroyEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cleanupTasksReply | [CleanupTasksReply](#o2control.CleanupTasksReply) |  |  |






<a name="o2control.DestroyEnvironmentRequest"></a>

### DestroyEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| keepTasks | [bool](#bool) |  |  |
| allowInRunningState | [bool](#bool) |  |  |
| force | [bool](#bool) |  |  |






<a name="o2control.Empty"></a>

### Empty







<a name="o2control.EnvironmentInfo"></a>

### EnvironmentInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| createdWhen | [string](#string) |  |  |
| state | [string](#string) |  |  |
| tasks | [ShortTaskInfo](#o2control.ShortTaskInfo) | repeated |  |
| rootRole | [string](#string) |  |  |
| currentRunNumber | [uint32](#uint32) |  |  |
| defaults | [EnvironmentInfo.DefaultsEntry](#o2control.EnvironmentInfo.DefaultsEntry) | repeated |  |
| vars | [EnvironmentInfo.VarsEntry](#o2control.EnvironmentInfo.VarsEntry) | repeated |  |
| userVars | [EnvironmentInfo.UserVarsEntry](#o2control.EnvironmentInfo.UserVarsEntry) | repeated |  |






<a name="o2control.EnvironmentInfo.DefaultsEntry"></a>

### EnvironmentInfo.DefaultsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.EnvironmentInfo.UserVarsEntry"></a>

### EnvironmentInfo.UserVarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.EnvironmentInfo.VarsEntry"></a>

### EnvironmentInfo.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.EnvironmentOperation"></a>

### EnvironmentOperation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [EnvironmentOperation.Optype](#o2control.EnvironmentOperation.Optype) |  |  |
| roleName | [string](#string) |  |  |






<a name="o2control.Ev_EnvironmentEvent"></a>

### Ev_EnvironmentEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environmentId | [string](#string) |  |  |
| state | [string](#string) |  |  |
| currentRunNumber | [uint32](#uint32) |  |  |
| error | [string](#string) |  |  |
| message | [string](#string) |  |  |






<a name="o2control.Ev_RoleEvent"></a>

### Ev_RoleEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| status | [string](#string) |  |  |
| state | [string](#string) |  |  |
| rolePath | [string](#string) |  |  |






<a name="o2control.Ev_TaskEvent"></a>

### Ev_TaskEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| taskid | [string](#string) |  |  |
| state | [string](#string) |  |  |
| status | [string](#string) |  |  |
| hostname | [string](#string) |  |  |
| className | [string](#string) |  |  |






<a name="o2control.Event"></a>

### Event



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timestamp | [string](#string) |  |  |
| environmentEvent | [Ev_EnvironmentEvent](#o2control.Ev_EnvironmentEvent) |  |  |
| taskEvent | [Ev_TaskEvent](#o2control.Ev_TaskEvent) |  |  |
| roleEvent | [Ev_RoleEvent](#o2control.Ev_RoleEvent) |  |  |






<a name="o2control.Event_MesosHeartbeat"></a>

### Event_MesosHeartbeat







<a name="o2control.GetEnvironmentPropertiesReply"></a>

### GetEnvironmentPropertiesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [GetEnvironmentPropertiesReply.PropertiesEntry](#o2control.GetEnvironmentPropertiesReply.PropertiesEntry) | repeated |  |






<a name="o2control.GetEnvironmentPropertiesReply.PropertiesEntry"></a>

### GetEnvironmentPropertiesReply.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.GetEnvironmentPropertiesRequest"></a>

### GetEnvironmentPropertiesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| queries | [string](#string) | repeated | If len(queries) == 0, we return an empty map. To retrieve all KVs, use query &#39;*&#39; |
| excludeGlobals | [bool](#bool) |  |  |






<a name="o2control.GetEnvironmentReply"></a>

### GetEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [EnvironmentInfo](#o2control.EnvironmentInfo) |  |  |
| workflow | [RoleInfo](#o2control.RoleInfo) |  |  |






<a name="o2control.GetEnvironmentRequest"></a>

### GetEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="o2control.GetEnvironmentsReply"></a>

### GetEnvironmentsReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| frameworkId | [string](#string) |  |  |
| environments | [EnvironmentInfo](#o2control.EnvironmentInfo) | repeated |  |






<a name="o2control.GetEnvironmentsRequest"></a>

### GetEnvironmentsRequest
Environment
//////////////////////////////////////






<a name="o2control.GetFrameworkInfoReply"></a>

### GetFrameworkInfoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| frameworkId | [string](#string) |  |  |
| environmentsCount | [int32](#int32) |  |  |
| tasksCount | [int32](#int32) |  |  |
| state | [string](#string) |  |  |
| hostsCount | [int32](#int32) |  |  |
| instanceName | [string](#string) |  |  |
| version | [Version](#o2control.Version) |  |  |






<a name="o2control.GetFrameworkInfoRequest"></a>

### GetFrameworkInfoRequest
Framework
//////////////////////////////////////






<a name="o2control.GetRolesReply"></a>

### GetRolesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [RoleInfo](#o2control.RoleInfo) | repeated |  |






<a name="o2control.GetRolesRequest"></a>

### GetRolesRequest
Roles
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| envId | [string](#string) |  |  |
| pathSpec | [string](#string) |  |  |






<a name="o2control.GetTaskReply"></a>

### GetTaskReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [TaskInfo](#o2control.TaskInfo) |  |  |






<a name="o2control.GetTaskRequest"></a>

### GetTaskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| taskId | [string](#string) |  |  |






<a name="o2control.GetTasksReply"></a>

### GetTasksReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tasks | [ShortTaskInfo](#o2control.ShortTaskInfo) | repeated |  |






<a name="o2control.GetTasksRequest"></a>

### GetTasksRequest







<a name="o2control.GetWorkflowTemplatesReply"></a>

### GetWorkflowTemplatesReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplates | [WorkflowTemplateInfo](#o2control.WorkflowTemplateInfo) | repeated |  |






<a name="o2control.GetWorkflowTemplatesRequest"></a>

### GetWorkflowTemplatesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repoPattern | [string](#string) |  |  |
| revisionPattern | [string](#string) |  |  |
| allBranches | [bool](#bool) |  |  |
| allTags | [bool](#bool) |  |  |






<a name="o2control.ListReposReply"></a>

### ListReposReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repos | [RepoInfo](#o2control.RepoInfo) | repeated |  |
| globalDefaultRevision | [string](#string) |  |  |






<a name="o2control.ListReposRequest"></a>

### ListReposRequest







<a name="o2control.ModifyEnvironmentReply"></a>

### ModifyEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| failedOperations | [EnvironmentOperation](#o2control.EnvironmentOperation) | repeated |  |
| id | [string](#string) |  |  |
| state | [string](#string) |  |  |






<a name="o2control.ModifyEnvironmentRequest"></a>

### ModifyEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| operations | [EnvironmentOperation](#o2control.EnvironmentOperation) | repeated |  |
| reconfigureAll | [bool](#bool) |  |  |






<a name="o2control.NewAutoEnvironmentReply"></a>

### NewAutoEnvironmentReply







<a name="o2control.NewAutoEnvironmentRequest"></a>

### NewAutoEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplate | [string](#string) |  |  |
| vars | [NewAutoEnvironmentRequest.VarsEntry](#o2control.NewAutoEnvironmentRequest.VarsEntry) | repeated |  |
| id | [string](#string) |  |  |






<a name="o2control.NewAutoEnvironmentRequest.VarsEntry"></a>

### NewAutoEnvironmentRequest.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.NewEnvironmentReply"></a>

### NewEnvironmentReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [EnvironmentInfo](#o2control.EnvironmentInfo) |  |  |






<a name="o2control.NewEnvironmentRequest"></a>

### NewEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| workflowTemplate | [string](#string) |  |  |
| vars | [NewEnvironmentRequest.VarsEntry](#o2control.NewEnvironmentRequest.VarsEntry) | repeated |  |






<a name="o2control.NewEnvironmentRequest.VarsEntry"></a>

### NewEnvironmentRequest.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.RefreshReposRequest"></a>

### RefreshReposRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control.RemoveRepoReply"></a>

### RemoveRepoReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| newDefaultRepo | [string](#string) |  |  |






<a name="o2control.RemoveRepoRequest"></a>

### RemoveRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control.RepoInfo"></a>

### RepoInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| default | [bool](#bool) |  |  |
| defaultRevision | [string](#string) |  |  |






<a name="o2control.RoleInfo"></a>

### RoleInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| status | [string](#string) |  |  |
| state | [string](#string) |  |  |
| fullPath | [string](#string) |  |  |
| taskIds | [string](#string) | repeated |  |
| roles | [RoleInfo](#o2control.RoleInfo) | repeated |  |
| defaults | [RoleInfo.DefaultsEntry](#o2control.RoleInfo.DefaultsEntry) | repeated |  |
| vars | [RoleInfo.VarsEntry](#o2control.RoleInfo.VarsEntry) | repeated |  |
| userVars | [RoleInfo.UserVarsEntry](#o2control.RoleInfo.UserVarsEntry) | repeated |  |






<a name="o2control.RoleInfo.DefaultsEntry"></a>

### RoleInfo.DefaultsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.RoleInfo.UserVarsEntry"></a>

### RoleInfo.UserVarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.RoleInfo.VarsEntry"></a>

### RoleInfo.VarsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.SetDefaultRepoRequest"></a>

### SetDefaultRepoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |






<a name="o2control.SetEnvironmentPropertiesReply"></a>

### SetEnvironmentPropertiesReply







<a name="o2control.SetEnvironmentPropertiesRequest"></a>

### SetEnvironmentPropertiesRequest
Environment, GET/SET properties
//////////////////////////////////////


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| properties | [SetEnvironmentPropertiesRequest.PropertiesEntry](#o2control.SetEnvironmentPropertiesRequest.PropertiesEntry) | repeated | If properties == nil, the core sets nothing and reply ok |






<a name="o2control.SetEnvironmentPropertiesRequest.PropertiesEntry"></a>

### SetEnvironmentPropertiesRequest.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.SetGlobalDefaultRevisionRequest"></a>

### SetGlobalDefaultRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision | [string](#string) |  |  |






<a name="o2control.SetRepoDefaultRevisionReply"></a>

### SetRepoDefaultRevisionReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| info | [string](#string) |  |  |






<a name="o2control.SetRepoDefaultRevisionRequest"></a>

### SetRepoDefaultRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| index | [int32](#int32) |  |  |
| revision | [string](#string) |  |  |






<a name="o2control.ShortTaskInfo"></a>

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
| deploymentInfo | [TaskDeploymentInfo](#o2control.TaskDeploymentInfo) |  |  |
| pid | [string](#string) |  |  |
| sandboxStdout | [string](#string) |  |  |






<a name="o2control.StatusReply"></a>

### StatusReply



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| state | [string](#string) |  |  |
| statusUpdates | [StatusUpdate](#o2control.StatusUpdate) | repeated |  |






<a name="o2control.StatusRequest"></a>

### StatusRequest
Global status
//////////////////////////////////////






<a name="o2control.StatusUpdate"></a>

### StatusUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [StatusUpdate.Level](#o2control.StatusUpdate.Level) |  |  |
| mesosHeartbeat | [Event_MesosHeartbeat](#o2control.Event_MesosHeartbeat) |  | TODO add other events here and in events.proto |






<a name="o2control.SubscribeRequest"></a>

### SubscribeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="o2control.TaskClassInfo"></a>

### TaskClassInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| controlMode | [string](#string) |  |  |






<a name="o2control.TaskDeploymentInfo"></a>

### TaskDeploymentInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hostname | [string](#string) |  |  |
| agentId | [string](#string) |  |  |
| offerId | [string](#string) |  |  |
| executorId | [string](#string) |  |  |






<a name="o2control.TaskInfo"></a>

### TaskInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| shortInfo | [ShortTaskInfo](#o2control.ShortTaskInfo) |  |  |
| classInfo | [TaskClassInfo](#o2control.TaskClassInfo) |  |  |
| inboundChannels | [ChannelInfo](#o2control.ChannelInfo) | repeated |  |
| outboundChannels | [ChannelInfo](#o2control.ChannelInfo) | repeated |  |
| commandInfo | [CommandInfo](#o2control.CommandInfo) |  |  |
| taskPath | [string](#string) |  |  |
| envId | [string](#string) |  |  |
| properties | [TaskInfo.PropertiesEntry](#o2control.TaskInfo.PropertiesEntry) | repeated |  |






<a name="o2control.TaskInfo.PropertiesEntry"></a>

### TaskInfo.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="o2control.TeardownReply"></a>

### TeardownReply







<a name="o2control.TeardownRequest"></a>

### TeardownRequest
Not implemented yet


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reason | [string](#string) |  |  |






<a name="o2control.Version"></a>

### Version



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| major | [int32](#int32) |  |  |
| minor | [int32](#int32) |  |  |
| patch | [int32](#int32) |  |  |
| build | [string](#string) |  |  |
| productName | [string](#string) |  |  |
| versionStr | [string](#string) |  |  |






<a name="o2control.WorkflowTemplateInfo"></a>

### WorkflowTemplateInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repo | [string](#string) |  |  |
| template | [string](#string) |  |  |
| revision | [string](#string) |  |  |





 


<a name="o2control.ControlEnvironmentRequest.Optype"></a>

### ControlEnvironmentRequest.Optype


| Name | Number | Description |
| ---- | ------ | ----------- |
| NOOP | 0 |  |
| START_ACTIVITY | 1 |  |
| STOP_ACTIVITY | 2 |  |
| CONFIGURE | 3 |  |
| RESET | 4 |  |
| GO_ERROR | 5 |  |



<a name="o2control.EnvironmentOperation.Optype"></a>

### EnvironmentOperation.Optype


| Name | Number | Description |
| ---- | ------ | ----------- |
| NOOP | 0 |  |
| REMOVE_ROLE | 3 |  |
| ADD_ROLE | 4 |  |



<a name="o2control.StatusUpdate.Level"></a>

### StatusUpdate.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEBUG | 0 |  |
| INFO | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |


 

 


<a name="o2control.Control"></a>

### Control


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| TrackStatus | [StatusRequest](#o2control.StatusRequest) | [StatusReply](#o2control.StatusReply) stream |  |
| GetFrameworkInfo | [GetFrameworkInfoRequest](#o2control.GetFrameworkInfoRequest) | [GetFrameworkInfoReply](#o2control.GetFrameworkInfoReply) |  |
| Teardown | [TeardownRequest](#o2control.TeardownRequest) | [TeardownReply](#o2control.TeardownReply) |  |
| GetEnvironments | [GetEnvironmentsRequest](#o2control.GetEnvironmentsRequest) | [GetEnvironmentsReply](#o2control.GetEnvironmentsReply) |  |
| NewAutoEnvironment | [NewAutoEnvironmentRequest](#o2control.NewAutoEnvironmentRequest) | [NewAutoEnvironmentReply](#o2control.NewAutoEnvironmentReply) |  |
| NewEnvironment | [NewEnvironmentRequest](#o2control.NewEnvironmentRequest) | [NewEnvironmentReply](#o2control.NewEnvironmentReply) |  |
| GetEnvironment | [GetEnvironmentRequest](#o2control.GetEnvironmentRequest) | [GetEnvironmentReply](#o2control.GetEnvironmentReply) |  |
| ControlEnvironment | [ControlEnvironmentRequest](#o2control.ControlEnvironmentRequest) | [ControlEnvironmentReply](#o2control.ControlEnvironmentReply) |  |
| ModifyEnvironment | [ModifyEnvironmentRequest](#o2control.ModifyEnvironmentRequest) | [ModifyEnvironmentReply](#o2control.ModifyEnvironmentReply) |  |
| DestroyEnvironment | [DestroyEnvironmentRequest](#o2control.DestroyEnvironmentRequest) | [DestroyEnvironmentReply](#o2control.DestroyEnvironmentReply) |  |
| GetTasks | [GetTasksRequest](#o2control.GetTasksRequest) | [GetTasksReply](#o2control.GetTasksReply) |  |
| GetTask | [GetTaskRequest](#o2control.GetTaskRequest) | [GetTaskReply](#o2control.GetTaskReply) |  |
| CleanupTasks | [CleanupTasksRequest](#o2control.CleanupTasksRequest) | [CleanupTasksReply](#o2control.CleanupTasksReply) |  |
| GetRoles | [GetRolesRequest](#o2control.GetRolesRequest) | [GetRolesReply](#o2control.GetRolesReply) |  |
| GetWorkflowTemplates | [GetWorkflowTemplatesRequest](#o2control.GetWorkflowTemplatesRequest) | [GetWorkflowTemplatesReply](#o2control.GetWorkflowTemplatesReply) |  |
| ListRepos | [ListReposRequest](#o2control.ListReposRequest) | [ListReposReply](#o2control.ListReposReply) |  |
| AddRepo | [AddRepoRequest](#o2control.AddRepoRequest) | [AddRepoReply](#o2control.AddRepoReply) |  |
| RemoveRepo | [RemoveRepoRequest](#o2control.RemoveRepoRequest) | [RemoveRepoReply](#o2control.RemoveRepoReply) |  |
| RefreshRepos | [RefreshReposRequest](#o2control.RefreshReposRequest) | [Empty](#o2control.Empty) |  |
| SetDefaultRepo | [SetDefaultRepoRequest](#o2control.SetDefaultRepoRequest) | [Empty](#o2control.Empty) |  |
| SetGlobalDefaultRevision | [SetGlobalDefaultRevisionRequest](#o2control.SetGlobalDefaultRevisionRequest) | [Empty](#o2control.Empty) |  |
| SetRepoDefaultRevision | [SetRepoDefaultRevisionRequest](#o2control.SetRepoDefaultRevisionRequest) | [SetRepoDefaultRevisionReply](#o2control.SetRepoDefaultRevisionReply) |  |
| Subscribe | [SubscribeRequest](#o2control.SubscribeRequest) | [Event](#o2control.Event) stream |  |

 



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

