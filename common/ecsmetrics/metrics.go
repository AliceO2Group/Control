package ecsmetrics

import (
	"fmt"
	internalmetrics "runtime/metrics"
	"time"

	"github.com/AliceO2Group/Control/common/monitoring"
)

var endRequestChannel chan struct{}

func gather() monitoring.Metric {
	samples := []internalmetrics.Sample{
		{Name: "/gc/cycles/total:gc-cycles"},
		{Name: "/memory/classes/other:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/sync/mutex/wait/total:seconds"},
		{Name: "/memory/classes/other:bytes"},
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/free:bytes"},
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
		{Name: "/memory/classes/heap/stacks:bytes"},
		{Name: "/memory/classes/heap/unused:bytes"},
	}

	// Collect metrics data
	internalmetrics.Read(samples)

	metric := NewMetric("golangruntimemetrics")

	for _, sample := range samples {
		switch sample.Value.Kind() {
		case internalmetrics.KindUint64:
			metric.AddValue(sample.Name, sample.Value.Uint64())
		case internalmetrics.KindFloat64:
			metric.AddValue(sample.Name, sample.Value.Float64())
		case internalmetrics.KindFloat64Histogram:
			fmt.Printf("Error: Histogram is not supported yet for metric [%s]", sample.Name)
			continue
		default:
			fmt.Printf("Unsupported kind %v for metric %s\n", sample.Value.Kind(), sample.Name)
			continue
		}
	}
	return metric
}

func StartGolangMetrics(period time.Duration) {
	go func() {
		for {
			select {
			case <-endRequestChannel:
				endRequestChannel <- struct{}{}
				return
			default:
				monitoring.Send(gather())
				time.Sleep(period)
			}
		}
	}()
}

func StopGolangMetrics() {
	endRequestChannel <- struct{}{}
	<-endRequestChannel
}
