#include "consumer.hpp"

namespace o2::kaki
{

KafkaConsumer::KafkaConsumer(const std::string& topic, const kafka::Properties& properties)
  : mConsumer{properties}
{
  mConsumer.subscribe({topic});
}

void KafkaConsumer::run(ProcessRecordsCallback cb)
{
  mIsRunning = true;
  while (mIsRunning) {
    const auto records = mConsumer.poll(std::chrono::milliseconds(100));
    if (!cb(records)) {
      mIsRunning = false;
    }
  }
}

bool KafkaConsumer::isRunning() const noexcept
{
  return mIsRunning;
}

void KafkaConsumer::stop() noexcept
{
  mIsRunning = false;
}

} // namespace o2::kaki
