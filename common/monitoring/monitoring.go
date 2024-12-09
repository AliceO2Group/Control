package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var (
	server  *http.Server
	metrics []Metric
	// channel that is used to request end of metrics server, it sends notification when server ended.
	// It needs to be read!!!
	endChannel chan struct{}

	// channel used to send metrics into the event loop
	metricsChannel chan Metric

	// channel for sending notifications to event loop that new http Request to report metrics arrived
	metricsRequestChannel chan struct{}

	// channel used to send metrics to be reported by http request from event loop
	metricsToRequest chan []Metric
)

func initChannels(messageBufferSize int) {
	endChannel = make(chan struct{})
	metricsRequestChannel = make(chan struct{})
	metricsChannel = make(chan Metric, messageBufferSize)
	metricsToRequest = make(chan []Metric)
}

func closeChannels() {
	close(endChannel)
	close(metricsRequestChannel)
	close(metricsChannel)
	close(metricsToRequest)
}

func eventLoop() {
	for {
		select {
		case <-metricsRequestChannel:
			shallowCopyMetrics := metrics
			metrics = make([]Metric, 0)
			metricsToRequest <- shallowCopyMetrics

		case metric := <-metricsChannel:
			metrics = append(metrics, metric)

		case <-endChannel:
			endChannel <- struct{}{}
			return
		}
	}
}

func exportMetricsAndReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metricsRequestChannel <- struct{}{}
	metricsToConvert := <-metricsToRequest
	if metricsToConvert == nil {
		metricsToConvert = make([]Metric, 0)
	}
	json.NewEncoder(w).Encode(metricsToConvert)
}

func Send(metric Metric) {
	metricsChannel <- metric
}

func handleFunc(endpointName string) {
	// recover is here to correctly allow multiple Starts and Stops of server
	defer func() {
		recover()
	}()

	http.HandleFunc(endpointName, exportMetricsAndReset)
}

// \param url In format if url:port to be used together with
// \param endpoint
func Start(port uint16, endpointName string, messageBufferSize int) error {
	if server != nil {
		return nil
	}

	initChannels(messageBufferSize)

	go eventLoop()

	server := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	handleFunc(endpointName)
	return server.ListenAndServe()
}

func Stop() {
	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	endChannel <- struct{}{}
	<-endChannel
	server = nil
}
