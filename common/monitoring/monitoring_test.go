/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package monitoring

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime/pprof"
	"strings"
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
	for len(metricsInternal.GetMetrics()) != requiredMessages {
		select {
		case <-timeoutChan:
			t.Errorf("Timeout %v triggered when waiting for %v messages, got %v", timeout, requiredMessages, len(metricsInternal.GetMetrics()))
			return

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSimpleStartStop(t *testing.T) {
	go Run(1234, "/random")
	isRunningWithTimeout(t, time.Second)
	Stop()
}

func TestStartMultipleStop(t *testing.T) {
	go Run(1234, "/random")
	isRunningWithTimeout(t, time.Second)
	Stop()
	Stop()
}

func cleaningUpAfterTest() {
	Stop()
}

func initTest() {
	go Run(12345, "notimportant")
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
		metric := &Metric{name: "test"}
		Send(metric)
		hasNumberOfMetrics(t, time.Second, 1)

		aggregatedMetrics := metricsInternal.GetMetrics()

		if aggregatedMetrics[0].name != "test" {
			t.Errorf("Got wrong name %s in stored metric", aggregatedMetrics[0].name)
		}
	})
}

func TestExportingMetrics(t *testing.T) {
	testFunction(t, func(t *testing.T) {
		metric := &Metric{name: "test"}
		Send(metric)
		hasNumberOfMetrics(t, time.Second, 1)

		metricsRequestedChannel <- struct{}{}
		metricsToExport := <-metricsExportedToRequest

		if len(metricsToExport) != 1 {
			t.Errorf("Got wrong amount of metrics %d, expected 1", len(metricsToExport))
		}

		if metricsToExport[0].name != "test" {
			t.Errorf("Got wrong name of metric %s, expected test", metricsToExport[0].name)
		}
	})
}

func TestHttpRun(t *testing.T) {
	go Run(9876, "/metrics")
	defer Stop()

	isRunningWithTimeout(t, time.Second)

	metric := Metric{name: "test"}
	metric.timestamp = time.Unix(10, 0)
	metric.AddTag("tag1", "42")
	metric.SetFieldInt64("value1", 11)
	Send(&metric)

	response, err := http.Get("http://localhost:9876/metrics")
	if err != nil {
		t.Fatalf("Failed to GET metrics at port 9876: %v", err)
	}
	message, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Failed to read response Body: %v", err)
	}

	receivedMetrics, err := parseMultipleLineProtocol(string(message))
	if err != nil {
		t.Errorf("Failed to parse message: %v", string(message))
	}

	receivedMetric := receivedMetrics[0]

	if receivedMetric.Name != "test" {
		t.Errorf("Got wrong name of metric %s, expected test", receivedMetric.Name)
	}

	if receivedMetric.Timestamp != time.Unix(10, 0).UnixNano() {
		t.Errorf("Got wrong timestamp of metric %d, expected 10", receivedMetric.Timestamp)
	}

	if len(receivedMetric.Tags) != 1 {
		t.Errorf("Got wrong number of tags %d, expected 1", len(receivedMetric.Tags))
	}

	if receivedMetric.Tags["tag1"] != "42" {
		t.Errorf("Failed to retreive tags: tag1 with value 42, %+v", receivedMetric.Tags)
	}

	if len(receivedMetric.Fields) != 1 {
		t.Errorf("Got wrong number of values %d, expected 1", len(receivedMetric.Fields))
	}

	if receivedMetric.Fields["value1"] != "11i" {
		t.Errorf("Failed to retreive tags: value1 with value 11: %+v", receivedMetric.Fields)
	}
}

func parseMultipleLineProtocol(input string) ([]struct {
	Name      string
	Tags      map[string]string
	Fields    map[string]string
	Timestamp int64
}, error,
) {
	lines := strings.Split(strings.TrimSpace(input), "\n")
	parsed := []struct {
		Name      string
		Tags      map[string]string
		Fields    map[string]string
		Timestamp int64
	}{}

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid line protocol format: %s", line)
		}

		// Parse measurement and tags
		nameTags := strings.Split(parts[0], ",")
		name := nameTags[0]
		tags := map[string]string{}
		for _, tag := range nameTags[1:] {
			kv := strings.SplitN(tag, "=", 2)
			if len(kv) == 2 {
				tags[kv[0]] = kv[1]
			}
		}

		// Parse fields (leave as raw string for comparison)
		fields := map[string]string{}
		for _, field := range strings.Split(parts[1], ",") {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) == 2 {
				fields[kv[0]] = kv[1]
			}
		}

		// Parse timestamp
		var ts int64
		_, err := fmt.Sscanf(parts[2], "%d", &ts)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp in line: %s", line)
		}

		parsed = append(parsed, struct {
			Name      string
			Tags      map[string]string
			Fields    map[string]string
			Timestamp int64
		}{
			Name:      name,
			Tags:      tags,
			Fields:    fields,
			Timestamp: ts,
		})
	}

	return parsed, nil
}

func MapsEqual[K comparable, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valueA := range a {
		if valueB, ok := b[key]; !ok || valueA != valueB {
			return false
		}
	}
	return true
}

func TestMetricsFormat(t *testing.T) {
	metrics := []Metric{}
	metric1 := Metric{name: "test"}
	metric1.timestamp = time.Unix(10, 0)
	metric1.AddTag("tag1", "42")
	metric1.SetFieldInt64("int64", 1)
	metric1.SetFieldUInt64("uint64", 2)
	metric1.SetFieldFloat64("float64", 3)

	metrics = append(metrics, metric1)

	metric2 := Metric{name: "test1"}
	metric2.timestamp = time.Unix(10, 0)
	metric2.AddTag("tag1", "43")
	metric2.SetFieldInt64("int64", 2)
	metric2.SetFieldUInt64("uint64", 3)
	metric2.SetFieldFloat64("float64", 4)

	metrics = append(metrics, metric2)

	buf := bytes.Buffer{}

	Format(&buf, metrics)

	metricsParsed, err := parseMultipleLineProtocol(buf.String())
	if err != nil {
		t.Error(err)
	}

	for idx, metricParsed := range metricsParsed {
		metricOrig := metrics[idx]

		if metricParsed.Name != metricOrig.name {
			t.Errorf("failed to compare %v and %v ", metricOrig, metricParsed)
		}
		// if !MapsEqual(metricParsed.Tags, metricOrig.Tags) {
		// 	t.Errorf("failed to compare %v and %v ", metricOrig.Tags, metricParsed.Tags)
		// }
		// if !MapsEqual(metricParsed.Fields, metricOrig.Values) {
		// 	t.Errorf("failed to compare %v and %v ", metricOrig.Tags, metricParsed.Tags)
		// }
		if metricParsed.Timestamp != metricOrig.timestamp.UnixNano() {
			t.Errorf("failed to compare %v and %v ", metricOrig.timestamp.UnixNano(), metricParsed.Timestamp)
		}
	}
}

// generate unix timestamp with given seconds and random amount of nsecs
func generateTimestamp(seconds int64, r *rand.Rand) time.Time {
	return time.Unix(seconds, r.Int63n(999999999))
}

func TestMetricsObject(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	tags1 := TagsType{Tag{"tag1", "1"}, Tag{"tag2", "2"}}
	metrics := NewMetricsAggregate()
	for i := 0; i != 1000; i++ {
		metrics.AddMetric(&Metric{name: "test", tags: tags1, fields: FieldsType{"val1": int64(1)}, timestamp: generateTimestamp(10, r)})
		metrics.AddMetric(&Metric{name: "test", tags: tags1, fields: FieldsType{"val1": int64(1)}, timestamp: generateTimestamp(11, r)})
		metrics.AddMetric(&Metric{name: "test", tags: tags1, fields: FieldsType{"val2": int64(1)}, timestamp: generateTimestamp(11, r)})
	}

	if len(metrics.metricsBuckets) != 2 {
		t.Errorf("there is diffent number of buckets, wanted 2 got %d", len(metrics.metricsBuckets))
	}

	var aggregatedVal1 int64
	var aggregatedVal2 int64
	for key, metric := range metrics.metricsBuckets {
		if key.timestamp != time.Unix(10, 0) && key.timestamp != time.Unix(11, 0) {
			t.Errorf("expected bucket timestamp as 10 or 11, got %ds, %dns", key.timestamp.Unix(), key.timestamp.UnixNano())
		}
		aggregatedVal1 += metric.fields["val1"].(int64)
		if _, ok := metric.fields["val2"]; ok {
			aggregatedVal2 += metric.fields["val2"].(int64)
		}
	}

	if aggregatedVal1 != 2000 {
		t.Errorf("all values \"val1\" in buckets inside the object should have been aggregated to 2000, but got %d", aggregatedVal1)
	}

	if aggregatedVal2 != 1000 {
		t.Errorf("all values \"val2\" in buckets inside the object should have been aggregated to 1000, but got %d", aggregatedVal2)
	}

	aggregatedMetrics := metrics.GetMetrics()
	if len(metrics.metricsBuckets) != 2 {
		t.Errorf("there should be only 2 metrics buckets, but the number is %d", len(metrics.metricsBuckets))
	}

	aggregatedVal1 = 0
	aggregatedVal2 = 0
	for _, metric := range aggregatedMetrics {
		if metric.timestamp != time.Unix(10, 0) && metric.timestamp != time.Unix(11, 0) {
			t.Errorf("expected aggregated metric timestamp as 10 or 11, got %ds, %dns", metric.timestamp.Unix(), metric.timestamp.UnixNano())
		}
		aggregatedVal1 += metric.fields["val1"].(int64)
		if _, ok := metric.fields["val2"]; ok {
			aggregatedVal2 += metric.fields["val2"].(int64)
		}
	}
	if aggregatedVal1 != 2000 {
		t.Errorf("all values \"val1\" of aggregated metrics should have been aggregated to 2000, but got %d", aggregatedVal1)
	}

	if aggregatedVal2 != 1000 {
		t.Errorf("all values \"val2\" of aggregated metrics should have been aggregated to 1000, but got %d", aggregatedVal2)
	}

	metrics.Clear()
	aggregatedMetrics = metrics.GetMetrics()
	if len(aggregatedMetrics) != 0 {
		t.Errorf("metrics object should be empty after clearing, but we got some metrics: %+v", aggregatedMetrics)
	}
}

func TestApproximateHistogram(t *testing.T) {
	histo := newReservoirSampling("test", 500)
	for i := 0; i != 20; i++ {
		histo.AddPoint(10)
	}

	mean, median, minimum, p10, p30, p70, p90, maximum, count, poolSize := histo.GetStats()

	if count != 20 {
		t.Errorf("wrong count before reset, expected 20, got %d", count)
	}

	if poolSize != 20 {
		t.Errorf("wrong poolSize, expected 20, got %d", poolSize)
	}

	if mean != 10 || median != 10 || minimum != 10 || p10 != 10 || p30 != 10 || p70 != 10 || p90 != 10 || maximum != 10 {
		t.Errorf("one of the values is not 10 when it should be 10, mean %v, median %v, 10p %v, 30p %v, 70p %v, 90p %v", mean, median, p10, p30, p70, p90)
	}

	histo.Reset()
	_, _, _, _, _, _, _, _, count, poolSize = histo.GetStats()
	if count != 0 {
		t.Errorf("wrong count before reset, expected 0, got %d", count)
	}

	if poolSize != 0 {
		t.Errorf("wrong poolSize, expected 0, got %d", poolSize)
	}

	for i := 0; i != 2000; i++ {
		histo.AddPoint(10)
	}

	mean, median, minimum, p10, p30, p70, p90, maximum, count, poolSize = histo.GetStats()

	if count != 2000 {
		t.Errorf("wrong count before reset, expected 2000, got %d", count)
	}

	if poolSize != 500 {
		t.Errorf("wrong poolSize, expected 500, got %d", poolSize)
	}

	if mean != 10 || median != 10 || minimum != 10 || p10 != 10 || p30 != 10 || p70 != 10 || p90 != 10 || maximum != 10 {
		t.Errorf("one of the values is not 10 when it should be 10, mean %v, median %v, 10p %v, 30p %v, 70p %v, 90p %v", mean, median, p10, p30, p70, p90)
	}

	histo.Reset()
	_, _, _, _, _, _, _, _, count, poolSize = histo.GetStats()
	if count != 0 {
		t.Errorf("wrong count before reset, expected 0, got %d", count)
	}

	if poolSize != 0 {
		t.Errorf("wrong poolSize, expected 0, got %d", poolSize)
	}

	for i := 0; i != 10000; i++ {
		histo.AddPoint((float64(rand.Int63n(100))))
	}

	mean, median, minimum, p10, p30, p70, p90, maximum, count, poolSize = histo.GetStats()

	if count != 10000 {
		t.Errorf("wrong count before reset, expected 10000, got %d", count)
	}

	if poolSize != 500 {
		t.Errorf("wrong poolSize, expected 500, got %d", poolSize)
	}

	if math.Abs(mean-50) > 10 {
		t.Errorf("wrong mean value, expected 50+-10 got %v", mean)
	}

	if math.Abs(float64(median-50)) > 10 {
		t.Errorf("wrong median value, expected 50+-10 got %v", median)
	}

	if float64(minimum) > 10 {
		t.Errorf("wrong min value, expected 0+-10 got %v", minimum)
	}

	if math.Abs(float64(p10-10)) > 10 {
		t.Errorf("wrong 10p value, expected 10+-10 got %v", p10)
	}

	if math.Abs(float64(p30-30)) > 10 {
		t.Errorf("wrong 30p value, expected 30+-10 got %v", p30)
	}

	if math.Abs(float64(p70-70)) > 10 {
		t.Errorf("wrong 50p value, expected 70+-10 got %v", p70)
	}

	if math.Abs(float64(p90-90)) > 10 {
		t.Errorf("wrong 90p value, expected 90+-10 got %v", p90)
	}

	if math.Abs(float64(maximum-100)) > 10 {
		t.Errorf("wrong max value, expected 100+-10 got %v", maximum)
	}
}

func TestMetricsHistogramObject(t *testing.T) {
	metricsHisto := NewMetricsReservoirSampling()
	r := rand.New(rand.NewSource(42))
	tags1 := TagsType{Tag{"tag1", "1"}, Tag{"tag2", "2"}}
	for i := 0; i != 2000; i++ {
		metricsHisto.AddMetric(&Metric{name: "test", tags: tags1, fields: FieldsType{"val1": int64(r.Int31n(100)), "val2": int64(r.Int31n(100))}, timestamp: generateTimestamp(10, r)})
		metricsHisto.AddMetric(&Metric{name: "test", tags: tags1, fields: FieldsType{"val1": int64(r.Int31n(100)), "val2": int64(r.Int31n(100))}, timestamp: generateTimestamp(15, r)})
	}
	metrics := metricsHisto.GetMetrics()
	if len(metrics) != 4 {
		t.Errorf("received wrong number of histogram metrics, expected 4, got %d", len(metrics))
	}

	for _, metric := range metrics {
		for valueName, value := range metric.fields {
			if strings.Contains(valueName, "mean") {
				if math.Abs(value.(float64)-50) > 10 {
					t.Errorf("wrong mean value, expected 50+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "median") {
				if math.Abs(value.(float64)-50) > 10 {
					t.Errorf("wrong median value, expected 50+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "p10") {
				if math.Abs(value.(float64)-10) > 10 {
					t.Errorf("wrong p10, expected 10+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "p30") {
				if math.Abs(value.(float64)-30) > 10 {
					t.Errorf("wrong p30, expected 30+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "p70") {
				if math.Abs(value.(float64)-70) > 10 {
					t.Errorf("wrong p70, expected 70+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "p90") {
				if math.Abs(value.(float64)-90) > 10 {
					t.Errorf("wrong p90, expected 90+-10 got %v", value.(float64))
				}
			}
			if strings.Contains(valueName, "count") {
				if value.(uint64) != 2000 {
					t.Errorf("wrong number of count, expected 2000 got %v", value.(uint64))
				}
			}
			if strings.Contains(valueName, "poolsize") {
				if value.(uint64) != 1000 {
					t.Errorf("wrong number of poolsize, expected 1000 got %v", value.(uint64))
				}
			}
		}
	}

	metricsHisto.Clear()
	if m := metricsHisto.GetMetrics(); len(m) != 0 {
		t.Errorf("histogram metrics should be empty after clearing, but we got histogram metrics: %+v", m)
	}
}

func measureFunc(metric *Metric) {
	defer Timer(metric, Millisecond)()
	defer Timer(metric, Nanosecond)()
	time.Sleep(100 * time.Millisecond)
}

func TestTimers(t *testing.T) {
	metric := NewMetric("test")
	measureFunc(&metric)
	fields := metric.fields
	if fields["execution_time_ms"].(int64) < 100 {
		t.Error("wrong milliseconds")
	}
	if fields["execution_time_ns"].(int64) < 100000000 {
		t.Error("wrong nanoseconds")
	}
}

func BenchmarkSimple(b *testing.B) {
	cpuProfileFile, err := os.Create("cpu_profile.pprof")
	if err != nil {
		b.Fatalf("could not create CPU profile: %v", err)
	}
	defer cpuProfileFile.Close()

	pprof.StartCPUProfile(cpuProfileFile)
	defer pprof.StopCPUProfile()

	metrics := NewMetricsAggregate()

	metricToAdd1Tag1Val := Metric{name: "metricToAdd1Tag1Val", tags: TagsType{Tag{"tag1", "tag1"}}, fields: FieldsType{"val1": int64(1)}, timestamp: time.Unix(10, 1111)}
	metricToAdd2Tag2Val := Metric{name: "metricToAdd1Tag1Val", tags: TagsType{Tag{"tag1", "tag1"}, Tag{"tag2", "tag2"}}, fields: FieldsType{"val1": int64(1), "val2": int64(2)}, timestamp: time.Unix(10, 1223)}
	metricToAdd3Tag3Val := Metric{name: "metricToAdd1Tag1Val", tags: TagsType{Tag{"tag1", "tag1"}, Tag{"tag2", "tag2"}, Tag{"tag3", "tag3"}}, fields: FieldsType{"val1": int64(10), "val2": int64(2), "val3": int64(3)}, timestamp: time.Unix(10, 2531)}

	for b.Loop() {
		metrics.AddMetric(&metricToAdd1Tag1Val)
		metrics.AddMetric(&metricToAdd2Tag2Val)
		metrics.AddMetric(&metricToAdd3Tag3Val)
	}

	heapProfileFile, err := os.Create("heap_profile.pprof")
	if err != nil {
		log.Fatalf("could not create heap profile: %v", err)
	}
	defer heapProfileFile.Close()
	pprof.WriteHeapProfile(heapProfileFile)
}

func BenchmarkSendingMetrics(b *testing.B) {
	go Run(12345, "/metrics")

	defer Stop()

	metric := Metric{name: "testname", timestamp: time.Unix(12345, 0)}
	metric.SetFieldInt64("value1", 42)
	metric.SetFieldInt64("value2", 42)
	metric.AddTag("tag1", "40")
	metric.AddTag("tag2", "40")

	for b.Loop() {
		Send(&metric)
	}
}
