package ecsmetrics

import (
	"time"

	"github.com/AliceO2Group/Control/common/monitoring"
)

func NewMetric(name string) monitoring.Metric {
	timestamp := time.Now()
	metric := monitoring.Metric{Name: name, Timestamp: timestamp.UnixMilli()}
	metric.AddTag("subsystem", "ECS")
	return metric
}

// Timer* functions are meant to be used with defer statement to measure runtime of given function:
// defer TimerNS(&metric)()
func TimerMS(metric *monitoring.Metric) func() {
	start := time.Now()
	return func() {
		metric.AddValue("execution_time_ms", time.Since(start).Milliseconds())
	}
}

func TimerNS(metric *monitoring.Metric) func() {
	start := time.Now()
	return func() {
		metric.AddValue("execution_time_ns", time.Since(start).Nanoseconds())
	}
}
