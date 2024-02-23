#ifndef O2_KAKI_CONSUMER_HPP
#define O2_KAKI_CONSUMER_HPP

#include <kafka/KafkaConsumer.h>

#include <atomic>
#include <functional>
#include <string>
#include <vector>

namespace o2::kaki
{

class KafkaConsumer
{
 public:
  // return false to stop processing
  using ProcessRecordsCallback = std::function<bool(const std::vector<kafka::clients::consumer::ConsumerRecord>&)>;

  // creates consumer a subscribe to given topic. Throws exceptions when fails // TODO better exception handling
  KafkaConsumer(const std::string& topic, const kafka::Properties& properties);

  // blocking call starts consuming given topic
  void run(ProcessRecordsCallback cb);
  bool isRunning() const noexcept;
  void stop() noexcept;

 private:
  kafka::clients::consumer::KafkaConsumer mConsumer;
  std::atomic_bool mIsRunning{false};
};

} // namespace o2::kaki

#endif
