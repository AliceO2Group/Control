package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// blocks until either IsRunning() returns true or timeout is triggered
func isRunningWithTimeout(t *testing.T, timeout time.Duration) {
	timeoutChan := time.After(timeout)
	for !IsRunning() {
		select {
		case <-timeoutChan:
			t.Errorf("Monitoring is not running even after %v", timeout)
			return

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// block until either length of metrics is the same as \requiredMessages or timeout is triggered
func hasNumberOfMetrics(t *testing.T, timeout time.Duration, requiredMessages int) {
	timeoutChan := time.After(timeout)
	for len(metrics) != requiredMessages {
		select {
		case <-timeoutChan:
			t.Errorf("Timeout %v triggered when waiting for %v messages, got %v", timeout, requiredMessages, len(metrics))
			return

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSimpleStartStop(t *testing.T) {
	go Run(1234, "/random", 100)
	isRunningWithTimeout(t, time.Second)
	Stop()
}

func TestStartMultipleStop(t *testing.T) {
	go Run(1234, "/random", 100)
	isRunningWithTimeout(t, time.Second)
	Stop()
	Stop()
}

func cleaningUpAfterTest() {
	Stop()
}

func initTest() {
	go Run(12345, "notimportant", 100)
}

// decorator function that properly inits and cleans after higher level test of Monitoring package
func testFunction(t *testing.T, testToRun func(*testing.T)) {
	initTest()
	isRunningWithTimeout(t, time.Second)
	testToRun(t)
	cleaningUpAfterTest()
}

func TestSendingSingleMetric(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metric := Metric{Name: "test"}
		Send(metric)
		hasNumberOfMetrics(t, time.Second, 1)

		if metrics[0].Name != "test" {
			t.Errorf("Got wrong name %s in stored metric", metrics[0].Name)
		}
	})
}

func TestExportingMetrics(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metric := Metric{Name: "test"}
		Send(metric)
		hasNumberOfMetrics(t, time.Second, 1)

		metricsRequestedChannel <- struct{}{}
		metricsToExport := <-metricsExportedToRequest

		if len(metricsToExport) != 1 {
			t.Errorf("Got wrong amount of metrics %d, expected 1", len(metricsToExport))
		}

		if metricsToExport[0].Name != "test" {
			t.Errorf("Got wrong name of metric %s, expected test", metricsToExport[0].Name)
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
		hasNumberOfMetrics(t, time.Second, 1)

		Send(metric)
		time.Sleep(100 * time.Millisecond)

		if len(metrics) != 1 {
			t.Errorf("Metrics length is %d, but should be 1 after sending second metric", len(metrics))
		}
	})
}

func TestHttpRun(t *testing.T) {
	go Run(9876, "/metrics", 10)
	defer Stop()

	isRunningWithTimeout(t, time.Second)

	metric := Metric{Name: "test"}
	metric.Timestamp = 10
	metric.AddTag("tag1", 42)
	metric.AddValue("value1", 11)
	Send(metric)

	response, err := http.Get("http://localhost:9876/metrics")
	if err != nil {
		t.Fatalf("Failed to GET metrics at port 9876: %v", err)
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
	Run(12345, "/metrics", 100)

	// this goroutine keeps clearing results so RAM does not exhausted
	go func() {
		for {
			select {
			case <-endChannel:
				endChannel <- struct{}{}
				break
			default:
				if len(metrics) >= 10000000 {
					metricsRequestedChannel <- struct{}{}
					<-metricsExportedToRequest
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
