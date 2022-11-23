# Workflow and Task Configuration

## Non-critical tasks

Any task in a workflow can be declared as non-critical. A non-critical task is a task that doesn't trigger a global environment ERROR in case of failure. The state of a non-critical task doesn't affect the environment state in any way.

To declare a task as non-critical, a line has to be added in the task role block within a workflow template file. Specifically, in the task section of such a task role (usually after the `load` statement), the line to add is `critical: false`, like in the following example:

```yaml
roles:
  - name: "non-critical-task"
    vars:
      non-critical-task-var: 'var-value'
    task:
      load: mytask
      critical: false
```

In the absence of an explicit `critical` trait for a given task role, the assumed default value is `critical: true`.