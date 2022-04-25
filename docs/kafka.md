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

As for today, AliECS publishes on two types of topics:

* `aliecs.env_state.<state>` where `state` can be `STANDBY`, `DEPLOYED`, `CONFIGURED`, `RUNNING`, `ERROR`, `UNKNOWN`.  For each topic, AliECS publishes a `NewStateNotification` message when any environment reaches the corresponding state. The `UNKNOWN` state is usually published when an environment gets a `DESTROY` request, but the plugin cannot know what will be the state after the transition.
* `aliecs.env_list.<state>` where `state` is only `RUNNING`. Each time there is an environment state change, AliECS publishes an `ActiveRunsList` message which contains a list of all environments which are currently in `RUNNING` state.

## Decoding the messages

Messages are encoded with protobuf. Please use [this](../core/integration/kafka/protos/kafka.proto) proto file to generate code which deserializes the messages.
