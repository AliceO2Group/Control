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

// Package monitoring provides monitoring and metrics collection functionality
// for O² Control components.
package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/sirupsen/logrus"
)

var (
	// atomic holder for the HTTP server instance
	server atomic.Pointer[http.Server]
	// objects to store incoming metrics
	metricsInternal          *MetricsAggregate         = NewMetricsAggregate()
	metricsHistogramInternal *MetricsReservoirSampling = NewMetricsReservoirSampling()
	// channel that is used to request end of metrics server, it sends notification when server ended.
	// It needs to be read!!!
	endChannel chan struct{} = make(chan struct{})

	// channel used to send metrics into the event loop
	// 100000 was chosen arbitrarily as a number that seemed sensible to be high enough to provide nice buffer if
	// multiple goroutines want to send metrics without blocking each other
	metricsChannel chan Metric = make(chan Metric, 100000)

	// channel used to send metrics meant to be proceesed as histogram into the event loop
	// 100000 was chosen arbitrarily as a number that seemed sensible to be high enough to provide nice buffer if
	// multiple goroutines want to send metrics without blocking each other
	metricsHistosChannel chan Metric = make(chan Metric, 100000)

	// channel for sending requests to reset actual metrics slice and send it back to caller via metricsExportedToRequest
	metricsRequestedChannel chan struct{} = make(chan struct{})

	// channel used to send metrics to be reported by http request from event loop
	metricsExportedToRequest chan []Metric = make(chan []Metric)

	// WaitUntilRunning is waiting until this channel is closed
	waitUntilRunningChannel chan struct{} = make(chan struct{})

	log = logger.New(logrus.StandardLogger(), "metrics").WithField(infologger.Level, infologger.IL_Devel)
)

// this eventLoop is the main part that processes all metrics send to the package
// 4 events can happen:
//  1. metricsChannel receives message from Send() method. We add the new metric to metrics slice
//  2. metricsHistosChannel receives message from Send() method. We add the new metric to metrics slice
//  3. metricsRequestChannel receives request to dump and request existing metrics. We send shallow copy of existing
//     metrics to requestor (via metricsExportedToRequest channel) while resetting current metrics slice
//  4. receive request to stop monitoring via endChannel. We send confirmation through endChannel to notify caller
//     that eventLoop stopped
func eventLoop() {
	for {
		select {
		case <-metricsRequestedChannel:
			aggregatedMetrics := metricsInternal.GetMetrics()
			aggregatedMetrics = append(aggregatedMetrics, metricsHistogramInternal.GetMetrics()...)
			metricsInternal.Clear()
			metricsHistogramInternal.Clear()

			metricsExportedToRequest <- aggregatedMetrics

		case metric := <-metricsChannel:
			metricsInternal.AddMetric(&metric)

		case metric := <-metricsHistosChannel:
			metricsHistogramInternal.AddMetric(&metric)

		case <-endChannel:
			defer func() {
				endChannel <- struct{}{}
			}()
			return
		}
	}
}

func exportMetricsAndReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	metricsRequestedChannel <- struct{}{}
	metricsToConvert := <-metricsExportedToRequest
	if metricsToConvert == nil {
		metricsToConvert = make([]Metric, 0)
	}
	err := Format(w, metricsToConvert)
	if err != nil {
		log.WithField(infologger.Level, infologger.IL_Devel).Errorf("Failed to export metrics: %v", err)
	}
}

func Send(metric *Metric) {
	// drop overflowing messages to not slowdown processing, we don't log so we don't flood IL
	select {
	case metricsChannel <- *metric:
	default:
	}
}

func SendHistogrammable(metric *Metric) {
	// drop overflowing messages to not slowdown processing, we don't log so we don't flood IL
	select {
	case metricsHistosChannel <- *metric:
	default:
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
//
// If we attempt send more messages than the size of the buffer, these overflowing messages will be ignored and warning will be logged.
func Run(port uint16, endpointName string) error {
	localServer := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	// only one Run should initialize and serve
	if !server.CompareAndSwap(nil, localServer) {
		return nil
	}
	go eventLoop()
	handleFunc(endpointName)
	// block until Shutdown is called
	close(waitUntilRunningChannel)
	return localServer.ListenAndServe()
}

func Stop() {
	localServer := server.Swap(nil)
	if localServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	localServer.Shutdown(ctx)
	endChannel <- struct{}{}
	_, ok := <-waitUntilRunningChannel
	if !ok {
		waitUntilRunningChannel = make(chan struct{})
	}
	<-endChannel
}

// If monitoring is not running it will wait until monitoring is running or
// timeout is triggered.
// \return true if monitoring is running, false if timeout occured
func WaitUntilRunning(timeout time.Duration) bool {
	timeoutChan := time.After(timeout)
	select {
	case <-waitUntilRunningChannel:
		return true
	case <-timeoutChan:
		return false
	}
}
