/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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
	//"flag"
	//"fmt"
	"time"

	"github.com/AliceO2Group/Control/common/product"
	"github.com/mesos/mesos-go/api/v1/cmd"
	"github.com/mesos/mesos-go/api/v1/lib/encoding/codecs"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FIXME: replace this with Viper
type Config struct {
	mesosFrameworkUser       string
	mesosFrameworkName       string
	mesosFrameworkRole       string
	mesosUrl                 string
	mesosCodec               codec
	mesosApiTimeout          time.Duration
	mesosFailoverTimeout     time.Duration
	mesosCheckpoint          bool
	mesosPrincipal           string
	mesosFrameworkHostname   string
	mesosLabels              Labels
	executor                 string
	verbose                  bool
	veryVerbose              bool
	executorCPU              float64
	executorMemory           float64
	mesosReviveBurst         int
	mesosReviveWait          time.Duration
	metrics                  metrics
	mesosResourceTypeMetrics bool
	mesosMaxRefuseSeconds    time.Duration
	mesosJobRestartDelay     time.Duration
	summaryMetrics           bool
	mesosExecutorImage       string
	mesosCompression         bool
	mesosCredentials         credentials
	mesosAuthMode            string
	mesosGpuClusterCompat    bool
	controlPort              int
	configurationUri         string
	instanceName             string
}

/*func (cfg *Config) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&cfg.mesosFrameworkUser, "mesos.framework.user", cfg.mesosFrameworkUser, "Framework user to register with the Mesos master")
	fs.StringVar(&cfg.mesosFrameworkName, "mesos.framework.name", cfg.mesosFrameworkName, "Framework name to register with the Mesos master")
	fs.StringVar(&cfg.mesosFrameworkRole, "mesos.framework.role", cfg.mesosFrameworkRole, "Framework role to register with the Mesos master")
	fs.Var(&cfg.mesosCodec, "mesos.codec", "Codec to encode/decode scheduler API communications [protobuf, json]")
	fs.StringVar(&cfg.mesosUrl, "mesos.url", cfg.mesosUrl, "Mesos scheduler API URL")
	fs.DurationVar(&cfg.mesosApiTimeout, "mesos.apiTimeout", cfg.mesosApiTimeout, "Mesos scheduler API connection timeout")
	fs.DurationVar(&cfg.mesosFailoverTimeout, "mesos.failoverTimeout", cfg.mesosFailoverTimeout, "Framework failover timeout (recover from scheduler failure)")
	fs.BoolVar(&cfg.mesosCheckpoint, "mesos.checkpoint", cfg.mesosCheckpoint, "Enable/disable agent checkpointing for framework tasks (recover from agent failure)")
	fs.StringVar(&cfg.mesosPrincipal, "mesos.principal", cfg.mesosPrincipal, "Framework principal with which to authenticate")
	fs.StringVar(&cfg.mesosFrameworkHostname, "mesos.frameworkHostname", cfg.mesosFrameworkHostname, "Framework hostname that is advertised to the master")
	fs.Var(&cfg.mesosLabels, "mesos.label", "Framework label, may be specified multiple times")
	fs.IntVar(&cfg.controlPort, "control.port", cfg.controlPort, "Port of control server")
	fs.StringVar(&cfg.executor, "executor.binary", cfg.executor, "Full path to executor binary on Mesos agents")
	fs.BoolVar(&cfg.verbose, "verbose", cfg.verbose, "Verbose logging")
	fs.BoolVar(&cfg.veryVerbose, "veryVerbose", cfg.veryVerbose, "Very verbose logging")
	fs.Float64Var(&cfg.executorCPU, "executor.cpu", cfg.executorCPU, "CPU resources to consume per-executor")
	fs.Float64Var(&cfg.executorMemory, "executor.memory", cfg.executorMemory, "Memory resources (MB) to consume per-executor")
	fs.IntVar(&cfg.mesosReviveBurst, "mesos.revive.burst", cfg.mesosReviveBurst, "Number of revive messages that may be sent in a burst within revive-wait period")
	fs.DurationVar(&cfg.mesosReviveWait, "mesos.revive.wait", cfg.mesosReviveWait, "Wait this long to fully recharge revive-burst quota")
	fs.StringVar(&cfg.metrics.address, "metrics.address", cfg.metrics.address, "IP of metrics server")
	fs.IntVar(&cfg.metrics.port, "metrics.port", cfg.metrics.port, "Port of metrics server (listens on server.address)")
	fs.StringVar(&cfg.metrics.path, "metrics.path", cfg.metrics.path, "URI path to metrics endpoint")
	fs.BoolVar(&cfg.summaryMetrics, "metrics.summary", cfg.summaryMetrics, "Collect summary metrics for tasks launched per-offer-cycle, offer processing time, etc.")
	fs.BoolVar(&cfg.mesosResourceTypeMetrics, "mesos.resourceTypeMetrics", cfg.mesosResourceTypeMetrics, "Collect scalar resource metrics per-type")
	fs.DurationVar(&cfg.mesosMaxRefuseSeconds, "mesos.maxRefuseSeconds", cfg.mesosMaxRefuseSeconds, "Max length of time to refuse future offers")
	fs.DurationVar(&cfg.mesosJobRestartDelay, "mesos.jobRestartDelay", cfg.mesosJobRestartDelay, "Duration between job (internal service) restarts between failures")
	fs.StringVar(&cfg.mesosExecutorImage, "executor.image", cfg.mesosExecutorImage, "Name of the docker image to run the executor")
	fs.BoolVar(&cfg.mesosCompression, "mesos.compression", cfg.mesosCompression, "When true attempt to use compression for HTTP streams.")
	fs.StringVar(&cfg.mesosCredentials.username, "mesos.credentials.username", cfg.mesosCredentials.username, "Username for Mesos authentication")
	fs.StringVar(&cfg.mesosCredentials.password, "mesos.credentials.passwordFile", cfg.mesosCredentials.password, "Path to file that contains the password for Mesos authentication")
	fs.StringVar(&cfg.mesosAuthMode, "mesos.authMode", cfg.mesosAuthMode, "Method to use for Mesos authentication; specify '"+AuthModeBasic+"' for simple HTTP authentication")
	fs.BoolVar(&cfg.mesosGpuClusterCompat, "mesos.gpuClusterCompat", cfg.mesosGpuClusterCompat, "When true the framework will receive offers from agents w/ GPU resources.")
	fs.StringVar(&cfg.configurationUri, "config", cfg.configurationUri, "URI of the Consul server or YAML configuration file.")
	fs.StringVar(&cfg.instanceName, "instance", cfg.instanceName, "User-visible name for this AliECS instance.")
}*/

/*func AddFlags(cfg *viper.Viper) {
	viper.Bind()
}*/

func setDefaults(cfg *viper.Viper) {
	exe, err := os.Executable() //TODO: Is this ok here?
	if err != nil {
		log.WithField("error", err).Error("cannot find scheduler executable path")
	}
	exeDir := filepath.Dir(exe)

	cfg.SetDefault("controlPort", 47102)
	cfg.SetDefault("executor", env("EXEC_BINARY", filepath.Join(exeDir, "o2control-executor")))
	cfg.SetDefault("executorCPU", envFloat("EXEC_CPU", "0.01"))
	cfg.SetDefault("executorMemory", envFloat("EXEC_MEMORY", "64"))
	cfg.SetDefault("instanceName", "AliECS instance")
	cfg.SetDefault("mesosApiTimeout", envDuration("MESOS_CONNECT_TIMEOUT", "20s"))
	cfg.SetDefault("mesosAuthMode", env("AUTH_MODE", ""))
	cfg.SetDefault("mesosCheckpoint", true)
	cfg.SetDefault("mesosCodec", codec{Codec: codecs.ByMediaType[codecs.MediaTypeProtobuf]})

	/*cfg.SetDefault("mesosCredentials", credentials{
		username: env("AUTH_USER", ""),
		password: env("AUTH_PASSWORD_FILE", ""),
	})*/
	cfg.SetDefault("mesosCredentials.username", env("AUTH_USER", ""))
	cfg.SetDefault("mesosCredentials.passwordFile", env("AUTH_PASSWORD_FILE", ""))

	cfg.SetDefault("mesosExecutorImage", env("EXEC_IMAGE", cmd.DockerImageTag))
	cfg.SetDefault("mesosFailoverTimeout", envDuration("SCHEDULER_FAILOVER_TIMEOUT", "1000h"))
	cfg.SetDefault("mesosFrameworkName", env("FRAMEWORK_NAME", product.NAME))
	cfg.SetDefault("mesosFrameworkUser", env("FRAMEWORK_USER", "root"))
	cfg.SetDefault("mesosJobRestartDelay", envDuration("JOB_RESTART_DELAY", "5s"))
	cfg.SetDefault("mesosMaxRefuseSeconds", envDuration("MAX_REFUSE_SECONDS", "5s"))
	cfg.SetDefault("mesosReviveBurst", envInt("REVIVE_BURST", "3"))
	cfg.SetDefault("mesosReviveWait", envDuration("REVIVE_WAIT", "1s"))
	cfg.SetDefault("mesosUrl", env("MESOS_MASTER_HTTP", "http://:5050/api/v1/scheduler"))

	/*cfg.SetDefault("metrics", metrics{
		address: env("LIBPROCESS_IP", "127.0.0.1"),
		port:    envInt("PORT0", "64009"),
		path:    env("METRICS_API_PATH", "/metrics"),
	})*/
	cfg.SetDefault("metrics.address", env("LIBPROCESS_IP", "127.0.0.1"))
	cfg.SetDefault("metrics.port", envInt("PORT0", "64009"))
	cfg.SetDefault("metrics.path", env("METRICS_API_PATH", "/metrics"))
}

func setFlags(cfg *viper.Viper) {
	pflag.String("executor", cfg.GetString("executor"), "Full path to executor binary on Mesos agents")
	pflag.String("mesosUrl", cfg.GetString("mesosUrl"), "Mesos scheduler API URL")
	pflag.String("mesosFrameworkUser", cfg.GetString("mesosFrameworkUser"), "Framework user to register with the Mesos master")
	pflag.String("mesosFrameworkName", cfg.GetString("mesosFrameworkName"), "Framework name to register with the Mesos master")
	pflag.String("mesosFrameworkRole", cfg.GetString("mesosFrameworkRole"), "Framework role to register with the Mesos master")
	//fs.Var(&cfg.mesosCodec, "mesos.codec", "Codec to encode/decode scheduler API communications [protobuf, json]")
	//pflag.Var(&mesosCodecObj, "mesosCodec","Framework role to register with the Mesos master")
	pflag.Duration("mesosApiTimeout", cfg.GetDuration("mesosApiTimeout"), "Mesos scheduler API connection timeout")
	pflag.Duration("mesosFailoverTimeout", cfg.GetDuration("mesosFailoverTimeout"), "Framework failover timeout (recover from scheduler failure)")
	pflag.Bool("mesosCheckpoint", cfg.GetBool("mesosCheckpoint"), "Enable/disable agent checkpointing for framework tasks (recover from agent failure)")
	pflag.String("mesosPrincipal", cfg.GetString("mesosPrincipal"), "Framework principal with which to authenticate")
	pflag.String("mesosFrameworkHostname", cfg.GetString("mesosFrameworkHostname"), "Framework hostname that is advertised to the master") // TODO: Is this the same as mesosFrameworkName??
	//fs.Var(&cfg.mesosLabels, "mesos.label", "Framework label, may be specified multiple times")
	//pflag.Var(&mesosLabelsObj, "mesosLabels", "Framework label, may be specified multiple times")
	pflag.Int("controlPort", cfg.GetInt("controlPort"), "Port of control server")
	pflag.Bool("verbose", cfg.GetBool("verbose"), "Verbose logging")
	pflag.Bool("veryVerbose", cfg.GetBool("veryVerbose"), "Very verbose logging")
	pflag.Float64("executorCPU", cfg.GetFloat64("executorCPU"), "CPU resources to consume per-executor")
	pflag.Float64("executorMemory", cfg.GetFloat64("executorMemory"), "Memory resources (MB) to consume per-executor")
	pflag.Int("mesosReviveBurst", cfg.GetInt("mesosReviveBurst"), "Number of revive messages that may be sent in a burst within revive-wait period")
	pflag.Duration("mesosReviveWait", cfg.GetDuration("mesosReviveWait"), "Wait this long to fully recharge revive-burst quota")

	//TODO: Make up mind concerning unmarshalling into a struct
	pflag.String("metrics.address", cfg.GetString("metrics.address"), "IP of metrics server")
	pflag.Int("metrics.port", cfg.GetInt("metrics.port"), "Port of metrics server (listens on server.address)")
	pflag.String("metrics.path", cfg.GetString("metrics.path"), "URI path to metrics endpoint")

	pflag.Bool("summaryMetrics", cfg.GetBool("summaryMetrics"), "Collect summary metrics for tasks launched per-offer-cycle, offer processing time, etc.")
	pflag.Bool("mesosResourceTypeMetrics", cfg.GetBool("mesosResourceTypeMetrics"), "Collect scalar resource metrics per-type")
	pflag.Duration("mesosMaxRefuseSeconds", cfg.GetDuration("mesosMaxRefuseSeconds"), "Max length of time to refuse future offers")
	pflag.Duration("mesosJobRestartDelay", cfg.GetDuration("mesosJobRestartDelay"), "Duration between job (internal service) restarts between failures")
	pflag.String("mesosExecutorImage", cfg.GetString("mesosExecutorImage"), "Name of the docker image to run the executor")
	pflag.Bool("mesosCompression", cfg.GetBool("mesosCompression"), "When true attempt to use compression for HTTP streams.")

	//TODO: Make up mind concerning unmarshalling into a struct
	pflag.String("mesosCredentials.username", cfg.GetString("mesosCredentials.username"), "Username for Mesos authentication")
	pflag.String("mesosCredentials.passwordFile", cfg.GetString("mesosCredentials.passwordFile"), "Path to file that contains the password for Mesos authentication")

	pflag.String("mesosAuthMode", cfg.GetString("mesosAuthMode"), "Method to use for Mesos authentication; specify '"+AuthModeBasic+"' for simple HTTP authentication")
	pflag.Bool("mesosGpuClusterCompat", cfg.GetBool("mesosGpuClusterCompat"), "When true the framework will receive offers from agents w/ GPU resources.")
	pflag.String("configurationUri", cfg.GetString("configurationUri"), "URI of the Consul server or YAML configuration file.")
	pflag.String("instanceName", cfg.GetString("instanceName"), "User-visible name for this AliECS instance.")

	pflag.Parse()
	cfg.BindPFlags(pflag.CommandLine)
	cfg.BindPFlag("metrics.path", pflag.Lookup("metrics.path"))
}

const AuthModeBasic = "basic"

// NewConfig is the constructor for a new config.
func NewConfig() viper.Viper {

	cfg := viper.New()
	setDefaults(cfg)
	setFlags(cfg)

	//TODO: Unmarshal into config struct (do we need it?)

	return *cfg
}

type metrics struct {
	address string
	port    int
	path    string
}

type credentials struct {
	username string
	password string
}
