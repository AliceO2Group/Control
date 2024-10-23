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
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/product"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/core/task/schedutil"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
)

func setDefaults() error {
	exe, err := os.Executable()
	if err != nil {
		return errors.New("cannot find scheduler executable path: " + err.Error())
	}
	exeDir := filepath.Dir(exe)

	viper.Set("component", "core")
	viper.SetDefault("version", false)
	viper.SetDefault("controlPort", 32102)
	viper.SetDefault("coreConfigurationUri", "")
	viper.SetDefault("consulBasePath", "o2/components/aliecs/ANY/any")
	viper.SetDefault("coreWorkingDir", "/var/lib/o2/aliecs")
	viper.SetDefault("defaultRepo", "github.com/AliceO2Group/ControlWorkflows/")
	viper.SetDefault("executor", getenv("EXEC_BINARY", filepath.Join(exeDir, "o2-aliecs-executor")))
	viper.SetDefault("executorCPU", getenvFloat("EXEC_CPU", "0.01"))
	viper.SetDefault("executorMemory", getenvFloat("EXEC_MEMORY", "64"))
	viper.SetDefault("globalDefaultRevision", "master")
	viper.SetDefault("instanceName", fmt.Sprintf("%s instance", product.PRETTY_SHORTNAME))
	viper.SetDefault("mesosApiTimeout", getenvDuration("MESOS_CONNECT_TIMEOUT", "20s"))
	viper.SetDefault("mesosAuthMode", getenv("AUTH_MODE", ""))
	viper.SetDefault("mesosCheckpoint", true)
	viper.SetDefault("mesosCompression", false)
	viper.SetDefault("mesosUseSystemProxy", false)
	viper.SetDefault("mesosCredentials.username", getenv("AUTH_USER", ""))
	viper.SetDefault("mesosCredentials.passwordFile", getenv("AUTH_PASSWORD_FILE", ""))
	viper.SetDefault("mesosExecutorImage", getenv("EXEC_IMAGE", ""))
	viper.SetDefault("mesosFailoverTimeout", getenvDuration("SCHEDULER_FAILOVER_TIMEOUT", "1000h"))
	viper.SetDefault("mesosFrameworkHostname", "")
	viper.SetDefault("mesosFrameworkName", getenv("FRAMEWORK_NAME", product.NAME))
	viper.SetDefault("mesosFrameworkRole", "")
	viper.SetDefault("mesosFrameworkUser", getenv("FRAMEWORK_USER", "root"))
	viper.SetDefault("mesosGpuClusterCompat", false)
	viper.SetDefault("mesosJobRestartDelay", getenvDuration("JOB_RESTART_DELAY", "5s"))
	viper.SetDefault("mesosLabels", schedutil.Labels{})
	viper.SetDefault("mesosMaxRefuseSeconds", getenvDuration("MAX_REFUSE_SECONDS", "5s"))
	viper.SetDefault("mesosPrincipal", "")
	viper.SetDefault("mesosReviveBurst", getenvInt("REVIVE_BURST", "3"))
	viper.SetDefault("mesosReviveWait", getenvDuration("REVIVE_WAIT", "1s"))
	viper.SetDefault("mesosResourceTypeMetrics", false)
	viper.SetDefault("mesosUrl", getenv("MESOS_MASTER_HTTP", "http://:5050/api/v1/scheduler"))
	viper.SetDefault("mesosCredentials.username", "")
	viper.SetDefault("mesosCredentials.passwordFile", "")
	viper.SetDefault("metrics.address", getenv("LIBPROCESS_IP", "127.0.0.1"))
	viper.SetDefault("metrics.port", getenvInt("PORT0", "64009"))
	viper.SetDefault("metrics.path", getenv("METRICS_API_PATH", "/metrics"))
	viper.SetDefault("reposSshKey", "")
	viper.SetDefault("summaryMetrics", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("veryVerbose", false)
	viper.SetDefault("dumpWorkflows", false)
	viper.SetDefault("configServiceUri", "apricot://127.0.0.1:32101")
	viper.SetDefault("bookkeepingBaseUri", "http://127.0.0.1:4000")
	viper.SetDefault("ccdbEndpoint", "http://ccdb-test.cern.ch:8080")
	viper.SetDefault("dcsServiceEndpoint", "//127.0.0.1:50051")
	viper.SetDefault("dcsServiceUseSystemProxy", false)
	viper.SetDefault("ddSchedulergRPCTimeout", "5s")
	viper.SetDefault("ddSchedulerEndpoint", "//127.0.0.1:50052")
	viper.SetDefault("ddSchedulerUseSystemProxy", false)
	viper.SetDefault("trgServiceEndpoint", "//127.0.0.1:50060")
	viper.SetDefault("trgPollingInterval", "3s")
	viper.SetDefault("trgPollingTimeout", "3s")
	viper.SetDefault("trgReconciliationTimeout", "5s")
	viper.SetDefault("odcEndpoint", "//127.0.0.1:50053")
	viper.SetDefault("odcPollingInterval", "3s")
	viper.SetDefault("odcUseSystemProxy", false)
	viper.SetDefault("testPluginEndpoint", "//127.0.0.1:00000")
	viper.SetDefault("integrationPlugins", []string{})
	viper.SetDefault("coreConfigEntry", "settings")
	viper.SetDefault("fmqPlugin", "OCClite")
	viper.SetDefault("fmqPluginSearchPath", "$CONTROL_OCCPLUGIN_ROOT/lib/")
	viper.SetDefault("bookkeepingToken", "token")
	viper.SetDefault("kafkaEndpoint", "localhost:9092")
	viper.SetDefault("concurrentWorkflowTemplateProcessing", true)
	viper.SetDefault("concurrentWorkflowTemplateIteratorProcessing", true)
	viper.SetDefault("concurrentIteratorRoleExpansion", true)
	viper.SetDefault("reuseUnlockedTasks", false)
	viper.SetDefault("configCache", true)
	viper.SetDefault("taskClassCacheTTL", 7*24*time.Hour)
	viper.SetDefault("kafkaEndpoints", []string{"localhost:9092"})
	viper.SetDefault("enableKafka", true)
	viper.SetDefault("logAllIL", false)
	return nil
}

func setFlags() error {
	pflag.Bool("version", viper.GetBool("version"), "The current AliECS core version")
	pflag.Int("controlPort", viper.GetInt("controlPort"), "Port of control server")
	pflag.String("coreConfigurationUri", viper.GetString("coreConfigurationUri"), "Consul URI or filesystem path to JSON/YAML configuration payload to initialize core settings [EXPERT SETTING]")
	pflag.String("coreWorkingDir", viper.GetString("coreWorkingDir"), "Path to a writable directory for runtime AliECS data")
	pflag.String("executor", viper.GetString("executor"), "Full path to executor binary on Mesos agents")
	pflag.Float64("executorCPU", viper.GetFloat64("executorCPU"), "CPU resources to consume per-executor")
	pflag.Float64("executorMemory", viper.GetFloat64("executorMemory"), "Memory resources (MB) to consume per-executor")
	pflag.String("instanceName", viper.GetString("instanceName"), "User-visible name for this AliECS instance")
	pflag.Duration("mesosApiTimeout", viper.GetDuration("mesosApiTimeout"), "Mesos scheduler API connection timeout")
	pflag.String("mesosAuthMode", viper.GetString("mesosAuthMode"), "Method to use for Mesos authentication; specify '"+schedutil.AuthModeBasic+"' for simple HTTP authentication")
	pflag.Bool("mesosCheckpoint", viper.GetBool("mesosCheckpoint"), "Enable/disable agent checkpointing for framework tasks (recover from agent failure)")
	pflag.Bool("mesosCompression", viper.GetBool("mesosCompression"), "When true attempt to use compression for HTTP streams")
	pflag.Bool("mesosUseSystemProxy", viper.GetBool("mesosUseSystemProxy"), "When true the https_proxy, http_proxy and no_proxy environment variables are obeyed")
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
	pflag.String("reposSshKey", viper.GetString("reposSshKey"), "Path to a readable private ssh key for repo operations")
	pflag.Bool("summaryMetrics", viper.GetBool("summaryMetrics"), "Collect summary metrics for tasks launched per-offer-cycle, offer processing time, etc.")
	pflag.Bool("verbose", viper.GetBool("verbose"), "Verbose logging")
	pflag.Bool("veryVerbose", viper.GetBool("veryVerbose"), "Very verbose logging")
	pflag.Bool("dumpWorkflows", viper.GetBool("dumpWorkflows"), "Dump unprocessed and processed workflow files (`$PWD/wf-{,un}processed-<timestamp>.json`)")
	pflag.String("configServiceUri", viper.GetString("configServiceUri"), "URI of the Apricot instance (`apricot://host:port`), Consul server (`consul://`) or YAML configuration file, entry point for all configuration")
	pflag.String("dcsServiceEndpoint", viper.GetString("dcsServiceEndpoint"), "Endpoint of the DCS gRPC service (`host:port`)")
	pflag.Bool("dcsServiceUseSystemProxy", viper.GetBool("dcsServiceUseSystemProxy"), "When true the https_proxy, http_proxy and no_proxy environment variables are obeyed")
	pflag.String("ddSchedulerEndpoint", viper.GetString("ddSchedulerEndpoint"), "Endpoint of the DD scheduler gRPC service (`host:port`)")
	pflag.Duration("ddSchedulergRPCTimeout", viper.GetDuration("ddSchedulergRPCTimeout"), "Timeout for gRPC calls in ddshed plugin")
	pflag.Bool("ddSchedulerUseSystemProxy", viper.GetBool("ddSchedulerUseSystemProxy"), "When true the https_proxy, http_proxy and no_proxy environment variables are obeyed")
	pflag.String("trgServiceEndpoint", viper.GetString("trgServiceEndpoint"), "Endpoint of the TRG gRPC service (`host:port`)")
	pflag.Duration("trgPollingInterval", viper.GetDuration("trgPollingInterval"), "How often to query the TRG gRPC service for run status (default: 3s)")
	pflag.Duration("trgPollingTimeout", viper.GetDuration("trgPollingTimeout"), "Timeout for the query to the TRG gRPC service for run status (default: 3s)")
	pflag.Duration("trgReconciliationTimeout", viper.GetDuration("trgReconciliationTimeout"), "Timeout for reconciliation requests to the TRG gRPC service (default: 5s)")
	pflag.String("odcEndpoint", viper.GetString("odcEndpoint"), "Endpoint of the ODC gRPC service (`host:port`)")
	pflag.String("odcPollingInterval", viper.GetString("odcPollingInterval"), "How often to query the ODC gRPC service for partition status (default: 3s)")
	pflag.Bool("odcUseSystemProxy", viper.GetBool("odcUseSystemProxy"), "When true the https_proxy, http_proxy and no_proxy environment variables are obeyed")
	pflag.String("testPluginEndpoint", viper.GetString("testPluginEndpoint"), "Endpoint of the TEST plugin, actually a NOOP")
	pflag.StringSlice("integrationPlugins", viper.GetStringSlice("integrationPlugins"), "List of integration plugins to load (default: empty)")
	pflag.String("coreConfigEntry", viper.GetString("coreConfigEntry"), "key for AliECS core configuration within the `aliecs` component [EXPERT SETTING]")
	pflag.String("fmqPlugin", viper.GetString("fmqPlugin"), "Name of the plugin for FairMQ tasks")
	pflag.String("fmqPluginSearchPath", viper.GetString("fmqPluginSearchPath"), "Path to the directory where the FairMQ plugins are found on controlled nodes")
	pflag.String("kafkaEndpoint", viper.GetString("kafkaEndpoint"), "Endpoint of the Kafka service (`host:port`)")
	pflag.String("bookkeepingBaseUri", viper.GetString("bookkeepingBaseUri"), "URI of the O² Bookkeeping service (`protocol://host:port`)")
	pflag.Bool("concurrentWorkflowTemplateProcessing", viper.GetBool("concurrentWorkflowTemplateProcessing"), "Process aggregators in workflow templates concurrently")
	pflag.Bool("concurrentWorkflowTemplateIteratorProcessing", viper.GetBool("concurrentWorkflowTemplateIteratorProcessing"), "Process iterators in workflow templates concurrently")
	pflag.Bool("concurrentIteratorRoleExpansion", viper.GetBool("concurrentIteratorRoleExpansion"), "Expand iterator roles concurrently during workflow template processing")
	pflag.Bool("reuseUnlockedTasks", viper.GetBool("reuseUnlockedTasks"), "Reuse unlocked active tasks when satisfying environment deployment requests")
	pflag.Bool("configCache", viper.GetBool("configCache"), "Enable cache layer between AliECS core and Apricot")
	pflag.Duration("taskClassCacheTTL", viper.GetDuration("taskClassCacheTTL"), "TTL for task class cache entries")
	pflag.StringSlice("kafkaEndpoints", viper.GetStringSlice("kafkaEndpoints"), "List of Kafka endpoints to connect to (default: localhost:9092)")
	pflag.Bool("enableKafka", viper.GetBool("enableKafka"), "Turn on the kafka messaging")
	pflag.Bool("logAllIL", viper.GetBool("logAllIL"), "Send all the logs into IL, including Debug and Trace messages")

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
	if len(coreCfgUri) != 0 {
		log.WithField("coreConfigurationUri", coreCfgUri).
			Warn("core configuration override (this should normally not happen)")
		uri, err := url.Parse(coreCfgUri)
		if err != nil {
			return err
		}
		if uri.Scheme == "file" {
			viper.SetConfigFile(uri.Host + uri.Path)
			if err := viper.ReadInConfig(); err != nil {
				return errors.New(coreCfgUri + ": " + err.Error())
			}
		} else if uri.Scheme == "consul" {
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
	}

	// aliecs configuration is assumed to live in aliecs/ANY/any/settings
	payload, err := apricot.Instance().GetComponentConfiguration(&componentcfg.Query{
		Component: "aliecs",
		RunType:   apricotpb.RunType_ANY,
		RoleName:  "any",
		EntryKey:  viper.GetString("coreConfigEntry"), // defaults to "settings"
	})
	if err != nil {
		return errors.New(viper.GetString("configServiceUri") + ": could not acquire core configuration (possibly bad configServiceUri, expecting apricot://*, consul://* or file://*)")
	}

	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(payload))); err != nil {
		return errors.New(coreCfgUri + ": " + err.Error())
	}
	return nil
}

func checkRepoDirRights() error {
	repoDir := filepath.Join(viper.GetString("coreWorkingDir"), "repos")
	utils.EnsureTrailingSlash(&repoDir)
	err := unix.Access(repoDir, unix.W_OK)
	if err != nil {
		return errors.New("No write access for configuration repositories path \"" + repoDir + "\": " + err.Error())
	}
	return nil
}

func checkWorkingDirRights() error {
	err := unix.Access(viper.GetString("coreWorkingDir"), unix.W_OK)
	if err != nil {
		return errors.New("No write access for core working path \"" + viper.GetString("coreWorkingDir") + "\": " + err.Error())
	}
	return nil
}

func checkSshKeyRights() error {
	reposSshKey := viper.GetString("reposSshKey")
	if reposSshKey == "" {
		log.Trace("No repos ssh key provided")
		return nil
	}

	err := unix.Access(reposSshKey, unix.R_OK)
	if err != nil {
		return errors.New("No read access for repos ssh key: \"" + viper.GetString("reposSshKey") + "\": " + err.Error())
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

	if viper.GetBool("version") {
		var executable string
		executable, err = os.Executable()
		if err != nil {
			executable = "unknown path"
			err = nil
		}
		var usr *user.User
		usr, err = user.Current()
		if err != nil {
			usr = &user.User{}
			err = nil
		}

		fmt.Printf("%s core (%s) v%s build %s running as user %s from %s\n",
			product.PRETTY_FULLNAME,
			product.PRETTY_SHORTNAME,
			product.VERSION,
			product.BUILD,
			usr.Username,
			executable)
		os.Exit(0)
		return
	}

	if err = parseCoreConfig(); err != nil {
		return
	}
	bindEnvironmentVariables()
	if err = checkFmqPluginName(); err != nil {
		return
	}
	if err = checkRepoDirRights(); err != nil {
		return
	}
	if err = checkSshKeyRights(); err != nil {
		log.Warning(err)
	}
	sanitizeWorkingPath()
	if err = checkWorkingDirRights(); err != nil {
		return
	}
	return
}
