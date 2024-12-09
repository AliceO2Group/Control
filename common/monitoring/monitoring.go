package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
)

var (
	server       *http.Server
	metricsLimit int = 1000000
	metrics      []Metric
	// channel that is used to request end of metrics server, it sends notification when server ended.
	// It needs to be read!!!
	endChannel chan struct{}

	// channel used to send metrics into the event loop
	metricsChannel chan Metric

	// channel for sending notifications to event loop that new http Request to report metrics arrived
	metricsRequestChannel chan struct{}

	// channel used to send metrics to be reported by http request from event loop
	metricsToRequest chan []Metric

	Log = logger.New(logrus.StandardLogger(), "metrics")
)

func initChannels(messageBufferSize int) {
	endChannel = make(chan struct{})
	metricsRequestChannel = make(chan struct{})
	metricsChannel = make(chan Metric, 100)
	metricsToRequest = make(chan []Metric)
	metricsLimit = messageBufferSize
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
			if len(metrics) < metricsLimit {
				metrics = append(metrics, metric)
			} else {
				Log.Warn("too many metrics waiting to be scraped. Are you sure that metrics scraping is running?")
			}

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

// \param port port where the scraping endpoint will be created
// \param endpointName name of the endpoint, which must start with a slash eg. "/internalmetrics"
// \param messageBufferSize size of buffer for messages where messages are kept between scraping request.
//
// If we attempt send more messages than the size of the buffer, these overflowing messages will be ignored and warning will be logged.
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
