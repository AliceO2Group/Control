# Workflow and Task Configuration

## Non-critical tasks

Any task in a workflow can be declared as non-critical. A non-critical task is a task that doesn't trigger a global environment ERROR in case of failure. The state of a non-critical task doesn't affect the environment state in any way.

To declare a task as non-critical, a line has to be added in the corresponding workflow template. Under the task role, in the `task` section (usually after the `load` statement), the line to add is `critical: false`, like in the following example:

```yaml
roles:
  - name: "non-critical-task"
    vars:
      non-critical-task-var: 'var-value'
    task:
      load: mytask
      critical: false
```