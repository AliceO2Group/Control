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
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/task/schedutil"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"golang.org/x/sys/unix"
)

func setDefaults() error {
	exe, err := os.Executable()
	if err != nil {
		return errors.New("Cannot find scheduler executable path: " + err.Error())
	}
	exeDir := filepath.Dir(exe)

	viper.SetDefault("controlPort", 47102)
	viper.SetDefault("coreConfigurationUri", "consul://127.0.0.1:8500") //TODO: TBD
	viper.SetDefault("consulBasePath", "o2/aliecs")
	viper.SetDefault("coreWorkingDir", "/var/lib/o2/aliecs")
	viper.SetDefault("defaultRepo", "github.com/AliceO2Group/ControlWorkflows/")
	viper.SetDefault("executor", env("EXEC_BINARY", filepath.Join(exeDir, "o2control-executor")))
	viper.SetDefault("executorCPU", envFloat("EXEC_CPU", "0.01"))
	viper.SetDefault("executorMemory", envFloat("EXEC_MEMORY", "64"))
	viper.SetDefault("globalDefaultRevision", "master")
	viper.SetDefault("instanceName", fmt.Sprintf("%s instance", product.PRETTY_SHORTNAME))
	viper.SetDefault("mesosApiTimeout", envDuration("MESOS_CONNECT_TIMEOUT", "20s"))
	viper.SetDefault("mesosAuthMode", env("AUTH_MODE", ""))
	viper.SetDefault("mesosCheckpoint", true)
	viper.SetDefault("mesosCompression", false)
	viper.SetDefault("mesosCredentials.username", env("AUTH_USER", ""))
	viper.SetDefault("mesosCredentials.passwordFile", env("AUTH_PASSWORD_FILE", ""))
	viper.SetDefault("mesosExecutorImage", env("EXEC_IMAGE", ""))
	viper.SetDefault("mesosFailoverTimeout", envDuration("SCHEDULER_FAILOVER_TIMEOUT", "1000h"))
	viper.SetDefault("mesosFrameworkHostname", "")
	viper.SetDefault("mesosFrameworkName", env("FRAMEWORK_NAME", product.NAME))
	viper.SetDefault("mesosFrameworkRole", "")
	viper.SetDefault("mesosFrameworkUser", env("FRAMEWORK_USER", "root"))
	viper.SetDefault("mesosGpuClusterCompat", false)
	viper.SetDefault("mesosJobRestartDelay", envDuration("JOB_RESTART_DELAY", "5s"))
	viper.SetDefault("mesosLabels", schedutil.Labels{})
	viper.SetDefault("mesosMaxRefuseSeconds", envDuration("MAX_REFUSE_SECONDS", "5s"))
	viper.SetDefault("mesosPrincipal", "")
	viper.SetDefault("mesosReviveBurst", envInt("REVIVE_BURST", "3"))
	viper.SetDefault("mesosReviveWait", envDuration("REVIVE_WAIT", "1s"))
	viper.SetDefault("mesosResourceTypeMetrics", false)
	viper.SetDefault("mesosUrl", env("MESOS_MASTER_HTTP", "http://:5050/api/v1/scheduler"))
	viper.SetDefault("mesosCredentials.username", "")
	viper.SetDefault("mesosCredentials.passwordFile", "")
	viper.SetDefault("metrics.address", env("LIBPROCESS_IP", "127.0.0.1"))
	viper.SetDefault("metrics.port", envInt("PORT0", "64009"))
	viper.SetDefault("metrics.path", env("METRICS_API_PATH", "/metrics"))
	viper.SetDefault("summaryMetrics", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("veryVerbose", false)
	viper.SetDefault("dumpWorkflows", false)
	viper.SetDefault("globalConfigurationUri", "") //TODO: TBD
	viper.SetDefault("fmqPlugin", "OCClite");
	viper.SetDefault("fmqPluginSearchPath", "$CONTROL_OCCPLUGIN_ROOT/lib/");
	return nil
}

func setFlags() error {
	pflag.Int("controlPort", viper.GetInt("controlPort"), "Port of control server")
	pflag.String("coreConfigurationUri", viper.GetString("coreConfigurationUri"), "URI of the Consul server or YAML configuration file, used for core configuration.")
	pflag.String("executor", viper.GetString("executor"), "Full path to executor binary on Mesos agents")
	pflag.Float64("executorCPU", viper.GetFloat64("executorCPU"), "CPU resources to consume per-executor")
	pflag.Float64("executorMemory", viper.GetFloat64("executorMemory"), "Memory resources (MB) to consume per-executor")
	pflag.String("instanceName", viper.GetString("instanceName"), "User-visible name for this AliECS instance.")
	pflag.Duration("mesosApiTimeout", viper.GetDuration("mesosApiTimeout"), "Mesos scheduler API connection timeout")
	pflag.String("mesosAuthMode", viper.GetString("mesosAuthMode"), "Method to use for Mesos authentication; specify '"+schedutil.AuthModeBasic+"' for simple HTTP authentication")
	pflag.Bool("mesosCheckpoint", viper.GetBool("mesosCheckpoint"), "Enable/disable agent checkpointing for framework tasks (recover from agent failure)")
	pflag.Bool("mesosCompression", viper.GetBool("mesosCompression"), "When true attempt to use compression for HTTP streams.")
	pflag.String("mesosExecutorImage", viper.GetString("mesosExecutorImage"), "Name of the docker image to run the executor")
	pflag.Duration("mesosFailoverTimeout", viper.GetDuration("mesosFailoverTimeout"), "Framework failover timeout (recover from scheduler failure)")
	pflag.String("mesosFrameworkHostname", viper.GetString("mesosFrameworkHostname"), "Framework hostname that is advertised to the master")
	pflag.String("mesosFrameworkName", viper.GetString("mesosFrameworkName"), "Framework name to register with the Mesos master")
	pflag.String("mesosFrameworkRole", viper.GetString("mesosFrameworkRole"), "Framework role to register with the Mesos master")
	pflag.String("mesosFrameworkUser", viper.GetString("mesosFrameworkUser"), "Framework user to register with the Mesos master")
	pflag.Bool("mesosGpuClusterCompat", viper.GetBool("mesosGpuClusterCompat"), "When true the framework will receive offers from agents w/ GPU resources.")
	pflag.Duration("mesosJobRestartDelay", viper.GetDuration("mesosJobRestartDelay"), "Duration between job (internal service) restarts between failures")
	pflag.Duration("mesosMaxRefuseSeconds", viper.GetDuration("mesosMaxRefuseSeconds"), "Max length of time to refuse future offers")
	pflag.String("mesosPrincipal", viper.GetString("mesosPrincipal"), "Framework principal with which to authenticate")
	pflag.Int("mesosReviveBurst", viper.GetInt("mesosReviveBurst"), "Number of revive messages that may be sent in a burst within revive-wait period")
	pflag.Duration("mesosReviveWait", viper.GetDuration("mesosReviveWait"), "Wait this long to fully recharge revive-burst quota")
	pflag.Bool("mesosResourceTypeMetrics", viper.GetBool("mesosResourceTypeMetrics"), "Collect scalar resource metrics per-type")
	pflag.String("mesosUrl", viper.GetString("mesosUrl"), "Mesos scheduler API URL")
	pflag.String("mesosCredentials.username", viper.GetString("mesosCredentials.username"), "Username for Mesos authentication")
	pflag.String("mesosCredentials.passwordFile", viper.GetString("mesosCredentials.passwordFile"), "Path to file that contains the password for Mesos authentication")
	pflag.String("metrics.address", viper.GetString("metrics.address"), "IP of metrics server")
	pflag.Int("metrics.port", viper.GetInt("metrics.port"), "Port of metrics server (listens on server.address)")
	pflag.String("metrics.path", viper.GetString("metrics.path"), "URI path to metrics endpoint")
	pflag.Bool("summaryMetrics", viper.GetBool("summaryMetrics"), "Collect summary metrics for tasks launched per-offer-cycle, offer processing time, etc.")
	pflag.Bool("verbose", viper.GetBool("verbose"), "Verbose logging")
	pflag.Bool("veryVerbose", viper.GetBool("veryVerbose"), "Very verbose logging")
	pflag.Bool("dumpWorkflows", viper.GetBool("dumpWorkflows"), "Dump unprocessed and processed workflow files (`$PWD/wf-{,un}processed-<timestamp>.json`)")
	pflag.String("globalConfigurationUri", viper.GetString("globalConfigurationUri"), "URI of the Consul server or YAML configuration file, used for global configuration.")
	pflag.String("fmqPlugin", viper.GetString("fmqPlugin"), "Name of the plugin for FairMQ tasks.")
	pflag.String("fmqPluginSearchPath", viper.GetString("fmqPluginSearchPath"), "Path to the directory where the FairMQ plugins are found on controlled nodes.")

	pflag.Parse()
	return viper.BindPFlags(pflag.CommandLine)
}

func checkFmqPluginName() error {
	allowedPluginNames := []string{"OCC", "OCClite"}
	chosenPlugin := viper.GetString("fmqPlugin")
	for _, allowed := range allowedPluginNames {
		if chosenPlugin == allowed {
			return nil
		}
	}
	return fmt.Errorf("plugin name \"%s\" is invalid, allowed values: %s",
		chosenPlugin,
		strings.Join(allowedPluginNames, ", "),
	)
}

func parseCoreConfig() error {
	coreCfgUri := viper.GetString("coreConfigurationUri")
	uri, err := url.Parse(coreCfgUri)
	if err != nil {
		return err
	}

	if uri.Scheme == "file" {
		viper.SetConfigFile(uri.Host + uri.Path)
		if err := viper.ReadInConfig(); err != nil {
			return errors.New(coreCfgUri + ": " + err.Error());
		}
	} else if uri.Scheme == "consul"{
		if err := viper.AddRemoteProvider("consul", uri.Host, uri.Path); err != nil {
			return err
		}
		viper.SetConfigType("yaml")
		if err := viper.ReadRemoteConfig(); err != nil {
			return errors.New(coreCfgUri + ": " + err.Error())
		}
	} else {
		return errors.New(coreCfgUri + ": Core configuration URI could not be parsed (Expecting consul://* or file://*)")
	}

	return nil
}

func checkRepoDirRights() error {
	repoDir := filepath.Join(viper.GetString("coreWorkingDir"),"repos")
	utils.EnsureTrailingSlash(&repoDir)
	err := unix.Access(repoDir, unix.W_OK)
	if err != nil {
		return errors.New("No write access for configuration repositories path \"" + repoDir + "\": "+ err.Error())
	}
	return nil
}

func checkWorkingDirRights() error {
	err := unix.Access(viper.GetString("coreWorkingDir"), unix.W_OK)
	if err != nil {
		return errors.New("No write access for core working path \"" + viper.GetString("coreWorkingDir") + "\": "+ err.Error())
	}
	return nil
}

// Remove trailing '/'
func sanitizeWorkingPath() {
	sanitizeWorkingPath := filepath.Clean(viper.GetString("coreWorkingDir"))
	viper.Set("coreWorkingDir", sanitizeWorkingPath)
}

// Bind environment variables with the prefix ALIECS
// e.g. ALIECS_EXECUTORCPU
func bindEnvironmentVariables() {
	viper.SetEnvPrefix("ALIECS")
	viper.AutomaticEnv()
}


// NewConfig is the constructor for a new config.
func NewConfig() (err error) {
	if err = setDefaults(); err != nil {
		return
	}
	if err = setFlags(); err != nil {
		return
	}
	if err = parseCoreConfig(); err != nil  {
		return
	}
	bindEnvironmentVariables()
	if err = checkFmqPluginName(); err != nil {
		return
	}
	if err = checkRepoDirRights(); err != nil {
		return
	}
	sanitizeWorkingPath()
	if err = checkWorkingDirRights(); err != nil {
		return
	}


	return
}
