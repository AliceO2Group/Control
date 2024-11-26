# Component Configuration

## Connectivity to controlled nodes

ECS relies on Mesos to know the state of the controlled nodes.
Thus, losing connection to a Mesos slave can be treated as a node being down or unresponsive.
In case a Mesos slave is lost, tasks belonging to it are set to ERROR state and treated as INACTIVE.
Then, the environment is transitioned to ERROR.

Mesos slave health check can be configured with `MESOS_MAX_AGENT_PING_TIMEOUTS` (`--max_agent_ping_timeouts`) and `MESOS_AGENT_PING_TIMEOUT` (`--agent_ping_timeout`) parameters for Mesos.
Effectively, the factor of the two parameters is the time needed to consider a slave/agent as lost.
Please refer to Mesos documentation for more details.