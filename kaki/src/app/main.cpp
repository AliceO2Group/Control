#include <kafka/KafkaConsumer.h>
#include <consumer.hpp>

#include <cstdlib>
#include <iostream>
#include <signal.h>
#include <string>

int main()
{
  using namespace kafka;
  using namespace kafka::clients::consumer;

  const std::string brokers = "127.0.0.1:9092";
  const Topic topic = "example-topic";

  // Prepare the configuration
  const Properties props({{"bootstrap.servers", {brokers}}});

  o2::kaki::kafkaConsumer consumer{topic, props};

  consumer.run([](const std::vector<kafka::clients::consumer::ConsumerRecord>& records) -> bool {
    for (const auto& record : records) {
      if (!record.error()) {
        std::cout << "Got a new message..." << std::endl;
        std::cout << "    Topic    : " << record.topic() << std::endl;
        std::cout << "    Partition: " << record.partition() << std::endl;
        std::cout << "    Offset   : " << record.offset() << std::endl;
        std::cout << "    Timestamp: " << record.timestamp().toString()
                  << std::endl;
        std::cout << "    Headers  : " << toString(record.headers())
                  << std::endl;
        std::cout << "    Key   [" << record.key().toString() << "]"
                  << std::endl;
        std::cout << "    Value [" << record.value().toString() << "]"
                  << std::endl;
      } else {
        std::cerr << record.toString() << std::endl;
      }
    }
    return true;
  });
}
