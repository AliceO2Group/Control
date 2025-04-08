package ecsmetrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/AliceO2Group/Control/common/monitoring"
)

func measureFunc(metric *monitoring.Metric) {
	defer TimerMS(metric)()
	defer TimerNS(metric)()
	time.Sleep(100 * time.Millisecond)
}

func TestSimpleStartStop(t *testing.T) {
	metric := NewMetric("test")
	measureFunc(&metric)
	fmt.Println(metric.Values["execution_time_ms"])
	fmt.Println(metric.Values["execution_time_ns"])
	if metric.Values["execution_time_ms"].(int64) < 100 {
		t.Error("wrong milliseconds")
	}
	if metric.Values["execution_time_ns"].(int64) < 100000000 {
		t.Error("wrong nanoseconds")
	}
}
