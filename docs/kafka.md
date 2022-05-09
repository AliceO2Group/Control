# Kafka plugin

The Kafka plugin in AliECS publishes updates messages about new states of environments and lists of environments in the RUNNING state.
The messages are encoded with protobuf.

## Making sure that AliECS sends messages

To enable the plugin, one should make sure that the following points are fullfiled.
* The consul instance includes coordinates to your kafka broker and enables the plugin.
  Navigate to `o2/components/aliecs/ANY/any/settings` and make sure the following key value pairs are there:
  ```
  kafkaEndpoint: "my-kafka-broker:9092"
  integrationPlugins: 
          - kafka
  ```
  Please restart the AliECS core if you modify this file.
* Plugin is enabled for the new environments. Make sure that there is a `true` value set in the consul instance at the path `o2/runtime/aliecs/vars/kafka_enabled`.
  Alternatively, one can put `kafka_enabled : true` in the Advanced configuration panel in the AliECS GUI.

## Currently available topics

As for today, AliECS publishes on the following types of topics:

* `aliecs.env_state.<state>` where `state` can be `STANDBY`, `DEPLOYED`, `CONFIGURED`, `RUNNING`, `ERROR`, `UNKNOWN`.  For each topic, AliECS publishes a `NewStateNotification` message when any environment reaches the corresponding state. The `UNKNOWN` state is usually published when an environment gets a `DESTROY` request, but the plugin cannot know what will be the state after the transition.
* `aliecs.env_leave_state.<state>` where `state` can be `STANDBY`, `DEPLOYED`, `CONFIGURED`, `RUNNING`, `ERROR`. For each topic, AliECS publishes a `NewStateNotification` message when any environment is about to leave the corresponding state.
* `aliecs.env_list.<state>` where `state` is only `RUNNING`. Each time there is an environment state change, AliECS publishes an `ActiveRunsList` message which contains a list of all environments which are currently in `RUNNING` state.

## Decoding the messages

Messages are encoded with protobuf. Please use [this](../core/integration/kafka/protos/kafka.proto) proto file to generate code which deserializes the messages.

## Getting Start of Run and End of Run notifications

To get SOR and EOR notifications, please subscribe to the two corresponding topics:
* `aliecs.env_state.RUNNING` for Start of Run
* `aliecs.env_leave_state.RUNNING` for End of Run

Both will provide `NewStateNotification` messages encoded with protobuf. Please note that the EOR message will still contain the RUNNING state, because it is sent just before the transition starts.

## Using Kafka debug tools

One can use some Kafka command line tools to verify that a given setup works correctly. One should make sure to have Kafka installed on the machine used to run the tools.

To get a list of topics:
```
/opt/kafka/kafka_2.12-2.8.1/bin/kafka-topics.sh --bootstrap-server <kafka_broker_host>:<port> --list
```

To subscribe to a concrete topic:
```
/opt/kafka/kafka_2.12-2.8.1/bin/kafka-console-consumer.sh --bootstrap-server <kafka_broker_host>:<port> --topic aliecs.env_state.RUNNING
```
Please note that Kafka is distributes the messages in the push-pull mode by default. Thus, if you subscribe to messages with a debug tool, you might not see them in another application.
