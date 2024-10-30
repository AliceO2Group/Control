/*
 * === This file is part of ALICE O² ===
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

package schedutil

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli"
	"github.com/mesos/mesos-go/api/v1/lib/httpcli/httpsched"
	"github.com/mesos/mesos-go/api/v1/lib/resources"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	proto "google.golang.org/protobuf/proto"
)

const AuthModeBasic = "basic"

var log = logger.New(logrus.StandardLogger(), "scheduler")

func PrepareExecutorInfo(
	execBinary, execImage string,
	wantsResources mesos.Resources,
	jobRestartDelay time.Duration,
) (*mesos.ExecutorInfo, error) {
	if execImage != "" {
		log.Trace("executor container image specified, will run")
		// Create mesos custom executor
		return &mesos.ExecutorInfo{
			Type:       mesos.ExecutorInfo_CUSTOM,
			ExecutorID: mesos.ExecutorID{Value: product.NAME + "-container-executor"},
			Name:       proto.String("AliECS container executor"),
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
		log.Trace("no executor container image specified, using executor binary")
		_, executorCmd := filepath.Split(execBinary)

		var (
			executorUris    = []mesos.CommandInfo_URI{}
			executorCommand = fmt.Sprintf("./%s", executorCmd)
		)
		executorUris = append(executorUris, mesos.CommandInfo_URI{Value: execBinary, Executable: proto.Bool(true)})

		// Create mesos custom executor
		return &mesos.ExecutorInfo{
			Type:       mesos.ExecutorInfo_CUSTOM,
			ExecutorID: mesos.ExecutorID{Value: product.NAME + "-executor"},
			Name:       proto.String("AliECS executor"),
			Command: &mesos.CommandInfo{
				Value: proto.String(executorCommand),
				URIs:  executorUris,
			},
			Resources: wantsResources,
		}, nil
	}
	return nil, errors.New("must specify an executor binary or image")
}

func BuildWantsExecutorResources(executorCPU float64, executorMemory float64) (r mesos.Resources) {
	r.Add(
		resources.NewCPUs(executorCPU).Resource,
		resources.NewMemory(executorMemory).Resource,
	)
	log.Trace("wants-executor-resources = " + r.String())
	return
}

type credentials struct {
	username string
	password string
}

func BuildHTTPSched(creds credentials) calls.Caller {
	var authConfigOpt httpcli.ConfigOpt
	// TODO(jdef) make this auth-mode configuration more pluggable
	if viper.GetString("mesosAuthMode") == AuthModeBasic {
		log.Println("configuring HTTP Basic authentication")
		// TODO(jdef) this needs testing once mesos 0.29 is available
		authConfigOpt = httpcli.BasicAuth(creds.username, creds.password)
	}
	mesosCodec := codecs.ByMediaType[codecs.MediaTypeProtobuf]
	cli := httpcli.New(
		httpcli.Endpoint(viper.GetString("mesosUrl")),
		httpcli.Codec(mesosCodec),
		httpcli.Do(httpcli.With(
			authConfigOpt,
			httpcli.Timeout(viper.GetDuration("mesosApiTimeout")),
			httpcli.Transport(func(transport *http.Transport) {
				if !viper.GetBool("mesosUseSystemProxy") {
					transport.Proxy = func(request *http.Request) (*url.URL, error) {
						return nil, nil
					}
				}
			}),
		)),
	)
	if viper.GetBool("mesosCompression") {
		// TODO(jdef) experimental; currently released versions of Mesos will accept this
		// header but will not send back compressed data due to flushing issues.
		log.Info("compression enabled")
		cli.With(httpcli.RequestOptions(httpcli.Header("Accept-Encoding", "gzip")))
	}
	return httpsched.NewCaller(cli)
}

func BuildFrameworkInfo() *mesos.FrameworkInfo {
	mesosCheckpoint := viper.GetBool("mesosCheckpoint")
	frameworkInfo := &mesos.FrameworkInfo{
		User:       viper.GetString("mesosFrameworkUser"),
		Name:       viper.GetString("mesosFrameworkName"),
		Checkpoint: &mesosCheckpoint,
	}
	failoverTimeout := viper.GetDuration("mesosFailoverTimeout").Seconds()
	if failoverTimeout > 0 {
		frameworkInfo.FailoverTimeout = &failoverTimeout
	}
	mesosFrameworkRole := viper.GetString("mesosFrameworkRole")
	if mesosFrameworkRole != "" {
		frameworkInfo.Role = &mesosFrameworkRole
	}
	mesosPrincipal := viper.GetString("mesosPrincipal")
	if mesosPrincipal != "" {
		frameworkInfo.Principal = &mesosPrincipal
	}
	mesosFrameworkHostname := viper.GetString("mesosFrameworkHostname")
	if mesosFrameworkHostname != "" {
		frameworkInfo.Hostname = &mesosFrameworkHostname
	}
	mesosLabels := viper.Get("mesosLabels").(Labels)
	if len(mesosLabels) > 0 {
		log.WithPrefix("scheduler").WithField("labels", mesosLabels).Debug("building frameworkInfo labels")
		frameworkInfo.Labels = &mesos.Labels{Labels: mesosLabels}
	}
	if viper.GetBool("mesosGpuClusterCompat") {
		frameworkInfo.Capabilities = append(frameworkInfo.Capabilities,
			mesos.FrameworkInfo_Capability{Type: mesos.FrameworkInfo_Capability_GPU_RESOURCES},
		)
	}
	return frameworkInfo
}
