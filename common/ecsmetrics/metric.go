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
