package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSimpleStartStop(t *testing.T) {
	go Start(1234, "/random", 100)
	time.Sleep(time.Millisecond * 100)
	Stop()
}

func TestStartMultipleStop(t *testing.T) {
	go Start(1234, "/random", 100)
	time.Sleep(time.Millisecond * 100)
	Stop()
	Stop()
}

func cleaningUpAfterTest() {
	endChannel <- struct{}{}
	<-endChannel
	closeChannels()
	metrics = make([]Metric, 0)
}

func initTest() {
	initChannels(100)
	// we need metrics channel to block so we don't end to quickly
	metricsChannel = make(chan Metric, 0)
	go eventLoop()
}

// decorator function that properly inits and cleans after higher level test of Monitoring package
func testFunction(t *testing.T, testToRun func(*testing.T)) {
	initTest()
	testToRun(t)
	cleaningUpAfterTest()
}

func TestSendingSingleMetric(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metric := Metric{Name: "test"}
		Send(metric)
		if len(metrics) != 1 {
			t.Error("wrong number of metrics, should be 1")
		}

		if metrics[0].Name != "test" {
			t.Errorf("Got wrong name %s in stored metric", metrics[0].Name)
		}
	})
}

func TestExportingMetrics(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metric := Metric{Name: "test"}
		Send(metric)

		metricsRequestChannel <- struct{}{}
		metrics := <-metricsToRequest

		if len(metrics) != 1 {
			t.Errorf("Got wrong amount of metrics %d, expected 1", len(metrics))
		}

		if metrics[0].Name != "test" {
			t.Errorf("Got wrong name of metric %s, expected test", metrics[0].Name)
		}
	})
}

func TestBufferLimit(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metricsLimit = 1
		metric := Metric{Name: "test"}
		metric.Timestamp = 10
		metric.AddTag("tag1", 42)
		metric.AddValue("value1", 11)
		Send(metric)

		if len(metrics) != 1 {
			t.Errorf("Metrics length is %d, but should be 1 after sending first metric", len(metrics))
		}

		Send(metric)
		time.Sleep(100 * time.Millisecond)

		if len(metrics) != 1 {
			t.Errorf("Metrics length is %d, but should be 1 after sending second metric", len(metrics))
		}
	})
}

func TestHttpRun(t *testing.T) {
	go Start(12345, "/metrics", 10)
	defer Stop()

	time.Sleep(time.Second)

	metric := Metric{Name: "test"}
	metric.Timestamp = 10
	metric.AddTag("tag1", 42)
	metric.AddValue("value1", 11)
	Send(metric)

	response, err := http.Get("http://localhost:12345/metrics")
	if err != nil {
		t.Fatalf("Failed to GET metrics at port 12345: %v", err)
	}
	decoder := json.NewDecoder(response.Body)
	var receivedMetrics []Metric
	if err = decoder.Decode(&receivedMetrics); err != nil {
		t.Fatalf("Failed to decoded Metric: %v", err)
	}

	receivedMetric := receivedMetrics[0]

	if receivedMetric.Name != "test" {
		t.Errorf("Got wrong name of metric %s, expected test", receivedMetric.Name)
	}

	if receivedMetric.Timestamp != 10 {
		t.Errorf("Got wrong timestamp of metric %d, expected 10", receivedMetric.Timestamp)
	}

	if len(receivedMetric.Tags) != 1 {
		t.Errorf("Got wrong number of tags %d, expected 1", len(receivedMetric.Tags))
	}

	if receivedMetric.Tags["tag1"].(float64) != 42 {
		t.Error("Failed to retreive tags: tag1 with value 42")
	}

	if len(receivedMetric.Values) != 1 {
		t.Errorf("Got wrong number of values %d, expected 1", len(receivedMetric.Values))
	}

	if receivedMetric.Values["value1"].(float64) != 11 {
		t.Error("Failed to retreive tags: value1 with value 11")
	}
}

// This benchmark cannot be run for too long as it will fill whole RAM even with
// results:
// goos: linux
// goarch: amd64
// pkg: github.com/AliceO2Group/Control/common/monitoring
// cpu: 11th Gen Intel(R) Core(TM) i9-11900H @ 2.50GHz
// BenchmarkSendingMetrics-16
//
// 123365481              192.6 ns/op
// PASS
// ok      github.com/AliceO2Group/Control/common/monitoring       44.686s
func BenchmarkSendingMetrics(b *testing.B) {
	Start(12345, "/metrics", 100)

	// this goroutine keeps clearing results so RAM does not exhausted
	go func() {
		for {
			select {
			case <-endChannel:
				endChannel <- struct{}{}
				break
			default:
				if len(metrics) >= 10000000 {
					metricsRequestChannel <- struct{}{}
					<-metricsToRequest
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	defer Stop()

	metric := Metric{Name: "testname", Timestamp: 12345}
	metric.AddValue("value", 42)
	metric.AddTag("tag", 40)

	for i := 0; i < b.N; i++ {
		Send(metric)
	}

	fmt.Println("")
}
