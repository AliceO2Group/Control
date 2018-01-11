/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2017 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * Portions from examples in <https://github.com/mesos/mesos-go>:
 *     Copyright 2013-2015, Mesosphere, Inc.
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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpsched"
	"github.com/mesos/mesos-go/api/v1/lib/resources"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
)


func prepareExecutorInfo(
	execBinary, execImage string,
	server server,
	wantsResources mesos.Resources,
	jobRestartDelay time.Duration,
	metricsAPI *metricsAPI,
) (*mesos.ExecutorInfo, error) {
	if execImage != "" {
		log.Println("Executor container image specified, will run")
		// Create mesos custom executor
		return &mesos.ExecutorInfo{
			Type:       mesos.ExecutorInfo_CUSTOM,
			ExecutorID: mesos.ExecutorID{Value: "octl-container-executor"},
			Name:       proto.String("O² container executor"),
			Command: &mesos.CommandInfo{
				Shell: func() *bool { x := false; return &x }(),
			},
			Container: &mesos.ContainerInfo{
				Type: mesos.ContainerInfo_DOCKER.Enum(),
				Docker: &mesos.ContainerInfo_DockerInfo{
					Image:          execImage,
					ForcePullImage: func() *bool { x := true; return &x }(),
					Parameters: []mesos.Parameter{
						{
							Key:   "entrypoint",
							Value: execBinary,
						}}}},
			Resources: wantsResources,
		}, nil
	} else if execBinary != "" {
		log.Println("No executor image specified, will serve executor binary from built-in HTTP server")
		listener, iport, err := newListener(server)
		if err != nil {
			return nil, err
		}
		server.port = iport // we're just working with a copy of server, so this is OK
		var (
			mux                    = http.NewServeMux()
			executorUris           = []mesos.CommandInfo_URI{}
			uri, executorCmd, err2 = serveExecutorArtifact(server, execBinary, mux)
			executorCommand        = fmt.Sprintf("./%s", executorCmd)
		)
		if err2 != nil {
			return nil, err2
		}
		wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metricsAPI.artifactDownloads()
			mux.ServeHTTP(w, r)
		})
		executorUris = append(executorUris, mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(true)})

		go forever("artifact-server", jobRestartDelay, metricsAPI.jobStartCount, func() error { return http.Serve(listener, wrapper) })
		log.Println("Serving executor artifacts...")

		// Create mesos custom executor
		return &mesos.ExecutorInfo{
			Type:       mesos.ExecutorInfo_CUSTOM,
			ExecutorID: mesos.ExecutorID{Value: "octl-executor"},
			Name:       proto.String("O² executor"),
			Command: &mesos.CommandInfo{
				Value: proto.String(executorCommand),
				URIs:  executorUris,
			},
			Resources: wantsResources,
		}, nil
	}
	return nil, errors.New("must specify an executor binary or image")
}

func buildWantsExecutorResources(config Config) (r mesos.Resources) {
	r.Add(
		resources.NewCPUs(config.execCPU).Resource,
		resources.NewMemory(config.execMemory).Resource,
	)
	log.Println("wants-executor-resources = " + r.String())
	return
}

func buildHTTPSched(cfg Config, creds credentials) calls.Caller {
	var authConfigOpt httpcli.ConfigOpt
	// TODO(jdef) make this auth-mode configuration more pluggable
	if cfg.authMode == AuthModeBasic {
		log.Println("configuring HTTP Basic authentication")
		// TODO(jdef) this needs testing once mesos 0.29 is available
		authConfigOpt = httpcli.BasicAuth(creds.username, creds.password)
	}
	cli := httpcli.New(
		httpcli.Endpoint(cfg.url),
		httpcli.Codec(cfg.codec.Codec),
		httpcli.Do(httpcli.With(
			authConfigOpt,
			httpcli.Timeout(cfg.timeout),
		)),
	)
	if cfg.compression {
		// TODO(jdef) experimental; currently released versions of Mesos will accept this
		// header but will not send back compressed data due to flushing issues.
		log.Println("compression enabled")
		cli.With(httpcli.RequestOptions(httpcli.Header("Accept-Encoding", "gzip")))
	}
	return httpsched.NewCaller(cli)
}

func buildFrameworkInfo(cfg Config) *mesos.FrameworkInfo {
	failoverTimeout := cfg.failoverTimeout.Seconds()
	frameworkInfo := &mesos.FrameworkInfo{
		User:       cfg.user,
		Name:       cfg.name,
		Checkpoint: &cfg.checkpoint,
	}
	if cfg.failoverTimeout > 0 {
		frameworkInfo.FailoverTimeout = &failoverTimeout
	}
	if cfg.role != "" {
		frameworkInfo.Role = &cfg.role
	}
	if cfg.principal != "" {
		frameworkInfo.Principal = &cfg.principal
	}
	if cfg.hostname != "" {
		frameworkInfo.Hostname = &cfg.hostname
	}
	if len(cfg.labels) > 0 {
		log.WithPrefix("scheduler").WithField("labels", cfg.labels).Debug("building frameworkInfo labels")
		frameworkInfo.Labels = &mesos.Labels{Labels: cfg.labels}
	}
	if cfg.gpuClusterCompat {
		frameworkInfo.Capabilities = append(frameworkInfo.Capabilities,
			mesos.FrameworkInfo_Capability{Type: mesos.FrameworkInfo_Capability_GPU_RESOURCES},
		)
	}
	return frameworkInfo
}

func loadCredentials(userConfig credentials) (result credentials, err error) {
	result = userConfig
	if result.password != "" {
		// this is the path to a file containing the password
		_, err = os.Stat(result.password)
		if err != nil {
			return
		}
		var f *os.File
		f, err = os.Open(result.password)
		if err != nil {
			return
		}
		defer f.Close()
		var bytes []byte
		bytes, err = ioutil.ReadAll(f)
		if err != nil {
			return
		}
		result.password = string(bytes)
	}
	return
}
