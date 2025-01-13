/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2020 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/AliceO2Group/Control/common/ecsmetrics"
	"github.com/AliceO2Group/Control/common/event/topic"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/monitoring"
	pb "github.com/AliceO2Group/Control/common/protos"
	"github.com/AliceO2Group/Control/core/the"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "core")

const (
	fileLimitWant = 65536
	fileLimitMin  = 8192
)

func parseMetricsEndpoint(metricsEndpoint string) (error, uint16, string) {
	pattern := `(^[0-9]{4,5})\/([a-zA-Z]+)`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(metricsEndpoint)

	if matches != nil {
		port, err := strconv.ParseUint(matches[1], 10, 16)
		if err != nil {
			return err, 0, ""
		}
		return nil, uint16(port), matches[2]
	} else {
		return errors.New("Failed to parse metrics endpoint: %s"), 0, ""
	}
}

func runMetrics() {
	metricsEndpoint := viper.GetString("metricsEndpoint")
	err, port, endpoint := parseMetricsEndpoint(metricsEndpoint)
	if err != nil {
		log.WithField("error", err).Error("Failed to parse metrics endpoint")
		return
	}

	go func() {
		log.Infof("Starting to listen on endpoint %s:%d for metrics", endpoint, port)
		if err := monitoring.Run(port, fmt.Sprintf("/%s", endpoint), viper.GetInt("metricsBufferSize")); err != nil && err != http.ErrServerClosed {
			ecsmetrics.StopGolangMetrics()
			log.Errorf("failed to run metrics on port %d and endpoint: %s")
		}
	}()

	ecsmetrics.StartGolangMetrics(10 * time.Second)
}

// Run is the entry point for this scheduler.
// TODO: refactor Config to reflect our specific requirements
func Run() error {
	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if viper.GetBool("veryVerbose") {
		logrus.SetLevel(logrus.TraceLevel)
	}

	if viper.GetBool("veryVerbose") {
		log.WithField("configuration", viper.AllSettings()).Debug("core starting up")
	}
	log.WithField("level", infologger.IL_Support).Infof("%s core (%s v%s build %s) starting up", product.PRETTY_FULLNAME, product.PRETTY_SHORTNAME, product.VERSION, product.BUILD)

	// We create a context and use its cancel func as a shutdown func to release
	// all resources. The shutdown func is stored in the scheduler.internalState.
	ctx, cancel := context.WithCancel(context.Background())

	// This only runs once to create a container for all data which comprises the
	// scheduler's state.
	// It also keeps count of the tasks launched/finished
	state, err := newGlobalState(cancel)
	if err != nil {
		return err
	}

	// Set up channel to receive Unix Signals
	signals(state)

	// Raise soft file limit
	err = setLimits()
	if err != nil {
		return err
	}

	// Start the Repo Manager instance
	log.WithField("level", infologger.IL_Support).Infof("Starting the Control Workflows repo manager")
	_ = the.RepoManager()

	// We now build the Control server
	s := NewServer(state)

	state.taskman.Start(ctx)

	// First message to Kafka
	the.EventWriterWithTopic(topic.Core).WriteEvent(&pb.Ev_MetaEvent_CoreStart{
		FrameworkId: state.taskman.GetFrameworkID(),
	})

	// Plugins need to start after taskman is running, because taskman provides the FID
	integration.PluginsInstance().InitAll(state.taskman.GetFrameworkID())
	runMetrics()
	defer ecsmetrics.StopGolangMetrics()
	defer monitoring.Stop()

	log.WithField("level", infologger.IL_Devel).Infof("Everything initiated and listening on control port: %d", viper.GetInt("controlPort"))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("controlPort")))
	if err != nil {
		log.WithField("error", err).
			WithField("port", viper.GetInt("controlPort")).
			Fatal("net.Listener failed to listen")
	}
	if err = s.Serve(lis); err != nil {
		log.WithField("error", err).Fatal("GRPC server failed to serve")
	}

	// Ensure all workflow plugins are destroyed before core teardown
	integration.PluginsInstance().DestroyAll()
	the.ClearEventWriters()

	return err
}

func setLimits() error {
	var rLimit syscall.Rlimit

	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	if rLimit.Cur > fileLimitWant {
		return nil
	}
	if rLimit.Max < fileLimitMin {
		err = fmt.Errorf("need at least %v file descriptors",
			fileLimitMin)
		return err
	}
	if rLimit.Max < fileLimitWant {
		rLimit.Cur = rLimit.Max
	} else {
		rLimit.Cur = fileLimitWant
	}
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		// try min value
		rLimit.Cur = fileLimitMin
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return err
		}
	}

	return nil
}

// end Run
