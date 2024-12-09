package ecsmetrics

import (
	internalmetrics "runtime/metrics"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/monitoring"
	"github.com/sirupsen/logrus"
)

var (
	endRequestChannel chan struct{}
	log               = logger.New(logrus.StandardLogger(), "ecsmetrics")
)

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
			log.Warning("Error: Histogram is not supported yet for metric [%s]", sample.Name)
			continue
		default:
			log.Warning("Unsupported kind %v for metric %s\n", sample.Value.Kind(), sample.Name)
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
