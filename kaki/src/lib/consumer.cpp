#include "consumer.hpp"

namespace o2::kaki
{

kafkaConsumer::kafkaConsumer(const std::string& topic, const kafka::Properties& properties)
  : mConsumer{properties}
{
  mConsumer.subscribe({topic});
}

void kafkaConsumer::run(ProcessRecordsCallback cb)
{
  mIsRunning = true;
  while (mIsRunning) {
    const auto records = mConsumer.poll(std::chrono::milliseconds(100));
    if (!cb(records)) {
      mIsRunning = false;
    }
  }
}

bool kafkaConsumer::isRunning() const noexcept
{
  return mIsRunning;
}

void kafkaConsumer::stop() noexcept
{
  mIsRunning = false;
}

} // namespace o2::kaki
