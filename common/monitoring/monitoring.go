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

	// channel for sending requests to reset actual metrics slice and send it back to caller via metricsExportedToRequest
	metricsRequestedChannel chan struct{}

	// channel used to send metrics to be reported by http request from event loop
	metricsExportedToRequest chan []Metric

	log = logger.New(logrus.StandardLogger(), "metrics")
)

func initChannels(messageBufferSize int) {
	endChannel = make(chan struct{})
	metricsRequestedChannel = make(chan struct{})
	// 100 was chosen arbitrarily as a number that seemed sensible to be high enough to provide nice buffer if
	// multiple goroutines want to send metrics without blocking each other
	metricsChannel = make(chan Metric, 10000)
	metricsExportedToRequest = make(chan []Metric)
	metricsLimit = messageBufferSize
}

func closeChannels() {
	close(endChannel)
	close(metricsRequestedChannel)
	close(metricsChannel)
	close(metricsExportedToRequest)
}

// this eventLoop is the main part that processes all metrics send to the package
// 3 events can happen:
//  1. metricsChannel receives message from Send() method. We just add the new metric to metrics slice
//  2. metricsRequestChannel receives request to dump and request existing metrics. We send shallow copy of existing
//     metrics to requestor (via metricsExportedToRequest channel) while resetting current metrics slice
//  3. receive request to stop monitoring via endChannel. We send confirmation through endChannel to notify caller
//     that eventLoop stopped
func eventLoop() {
	for {
		select {
		case <-metricsRequestedChannel:
			shallowCopyMetrics := metrics
			metrics = make([]Metric, 0)
			metricsExportedToRequest <- shallowCopyMetrics

		case metric := <-metricsChannel:
			if len(metrics) < metricsLimit {
				metrics = append(metrics, metric)
			} else {
				log.Warn("too many metrics waiting to be scraped. Are you sure that metrics scraping is running?")
			}

		case <-endChannel:
			defer func() {
				endChannel <- struct{}{}
			}()
			return
		}
	}
}

func exportMetricsAndReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	metricsRequestedChannel <- struct{}{}
	metricsToConvert := <-metricsExportedToRequest
	if metricsToConvert == nil {
		metricsToConvert = make([]Metric, 0)
	}
	json.NewEncoder(w).Encode(metricsToConvert)
}

func Send(metric Metric) {
	if IsRunning() {
		metricsChannel <- metric
	}
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
func Run(port uint16, endpointName string, messageBufferSize int) error {
	if IsRunning() {
		return nil
	}

	initChannels(messageBufferSize)

	go eventLoop()

	server = &http.Server{Addr: fmt.Sprintf(":%d", port)}
	handleFunc(endpointName)
	return server.ListenAndServe()
}

func Stop() {
	if !IsRunning() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	endChannel <- struct{}{}
	<-endChannel
	server = nil
	metrics = nil
}

func IsRunning() bool {
	return server != nil
}
