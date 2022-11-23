/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021-2022 CERN and copyright holders of ALICE O².
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

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. protos/odc.proto

package odc

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/common/gera"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	odc "github.com/AliceO2Group/Control/core/integration/odc/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	ODC_DIAL_TIMEOUT                = 2 * time.Second
	ODC_GENERAL_OP_TIMEOUT          = 5 * time.Second
	ODC_CONFIGURE_TIMEOUT           = 60 * time.Second
	ODC_PARTITIONINITIALIZE_TIMEOUT = 60 * time.Second
	ODC_START_TIMEOUT               = 15 * time.Second
	ODC_STOP_TIMEOUT                = 15 * time.Second
	ODC_RESET_TIMEOUT               = 30 * time.Second
	ODC_PARTITIONTERMINATE_TIMEOUT  = 30 * time.Second
	ODC_PADDING_TIMEOUT             = 3 * time.Second
	ODC_STATUS_TIMEOUT              = 3 * time.Second
	ODC_POLLING_INTERVAL            = 3 * time.Second
)

type Plugin struct {
	odcHost string
	odcPort int

	odcClient *RpcClient

	cachedStatus           *OdcStatus
	cachedStatusMu         sync.RWMutex
	cachedStatusCancelFunc context.CancelFunc
}

type OdcStatus struct {
	Partitions map[uid.ID]OdcPartitionInfo
	Status     odc.ReplyStatus
	Message    string
	Error      *odc.Error
}

type OdcPartitionInfo struct {
	RunNumber uint32
	State     string
}

func NewPlugin(endpoint string) integration.Plugin {
	u, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	portNumber, _ := strconv.Atoi(u.Port())

	return &Plugin{
		odcHost:   u.Hostname(),
		odcPort:   portNumber,
		odcClient: nil,
	}
}

func (p *Plugin) GetName() string {
	return "odc"
}

func (p *Plugin) GetPrettyName() string {
	return "ODC (EPN subcontrol)"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("odcEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.odcClient == nil {
		return "UNKNOWN"
	}
	return p.odcClient.conn.GetState().String()
}

func (p *Plugin) queryPartitionStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), ODC_STATUS_TIMEOUT)
	defer cancel()

	statusRep := &odc.StatusReply{}
	var err error

	statusRep, err = p.odcClient.Status(ctx, &odc.StatusRequest{Running: true}, grpc.EmptyCallOption{})
	if err != nil {
		log.WithField("level", infologger.IL_Support).
			WithField("call", "Status").
			WithError(err).Error("ODC error")
	}
	if statusRep == nil {
		log.WithField("level", infologger.IL_Support).
			WithField("call", "Status").
			WithError(fmt.Errorf("ODC Status response is nil")).Error("ODC error")
		statusRep = &odc.StatusReply{}
	}

	response := &OdcStatus{
		Status:     statusRep.Status,
		Message:    statusRep.Msg,
		Error:      statusRep.Error,
		Partitions: make(map[uid.ID]OdcPartitionInfo),
	}
	for _, v := range statusRep.Partitions {
		var id uid.ID
		id, err = uid.FromString(v.Partitionid)
		if err != nil {
			continue
		}
		response.Partitions[id] = OdcPartitionInfo{
			RunNumber: uint32(v.Runnr),
			State:     v.State,
		}
	}

	p.cachedStatusMu.Lock()
	p.cachedStatus = response
	p.cachedStatusMu.Unlock()
}

func (p *Plugin) GetData(_ []any) string {
	if p == nil || p.odcClient == nil {
		return ""
	}

	p.cachedStatusMu.RLock()
	r := p.cachedStatus
	if r == nil {
		p.cachedStatusMu.RUnlock()
		return ""
	}

	partitionStates := make(map[string]string)

	if r.Status == odc.ReplyStatus_SUCCESS {
		for id, partitionInfo := range r.Partitions {
			partitionStates[id.String()] = partitionInfo.State
		}
	}
	p.cachedStatusMu.RUnlock()

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) GetEnvironmentsData(envIds []uid.ID) map[uid.ID]string {
	if p == nil || p.odcClient == nil {
		return nil
	}

	p.cachedStatusMu.RLock()
	defer p.cachedStatusMu.RUnlock()

	if p.cachedStatus == nil {
		return nil
	}

	out := make(map[uid.ID]string)
	for _, id := range envIds {
		partitionInfo, ok := p.cachedStatus.Partitions[id]
		if !ok {
			continue
		}

		partitionInfoOut, err := json.Marshal(partitionInfo)
		if err != nil {
			continue
		}
		out[id] = string(partitionInfoOut[:])
	}

	return out
}

func (p *Plugin) Init(_ string) error {
	if p.odcClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.odcClient = NewClient(cxt, cancel, viper.GetString("odcEndpoint"))
		if p.odcClient == nil {
			return fmt.Errorf("failed to connect to ODC service on %s", viper.GetString("ddSchedulerEndpoint"))
		}
		log.Debug("ODC plugin initialized")
	}

	var ctx context.Context
	ctx, p.cachedStatusCancelFunc = context.WithCancel(context.Background())

	odcPollingIntervalStr := viper.GetString("odcPollingInterval")
	odcPollingInterval, err := time.ParseDuration(odcPollingIntervalStr)
	if err != nil {
		odcPollingInterval = ODC_POLLING_INTERVAL
		log.Debugf("ODC plugin cannot acquire polling interval, defaulting to %s", ODC_POLLING_INTERVAL.String())
	}

	// polling
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(odcPollingInterval):
				p.queryPartitionStatus()
			}
		}
	}()
	return nil
}

func (p *Plugin) ObjectStack(varStack map[string]string) (stack map[string]interface{}) {
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("ObjectStack cannot acquire environment ID")
		return
	}

	var csErr error
	configStack := apricot.Instance().GetDefaults()
	configStack, csErr = gera.MakeStringMapWithMap(apricot.Instance().GetVars()).WrappedAndFlattened(gera.MakeStringMapWithMap(configStack))
	if csErr != nil {
		log.Error("cannot access AliECS workflow configuration defaults")
		return
	}

	stack = make(map[string]interface{})
	stack["GenerateEPNWorkflowScript"] = func() (out string) {
		/*
			OCTRL-558 example:
			GEN_TOPO_HASH=[0/1] GEN_TOPO_SOURCE=[...] DDMODE=[TfBuilder Mode] GEN_TOPO_LIBRARY_FILE=[...]
			GEN_TOPO_WORKFLOW_NAME=[...] WORKFLOW_DETECTORS=[...] WORKFLOW_DETECTORS_QC=[...]
			WORKFLOW_DETECTORS_CALIB=[...] WORKFLOW_PARAMETERS=[...] RECO_NUM_NODES_OVERRIDE=[...]
			MULTIPLICITY_FACTOR_RAWDECODERS=[...] MULTIPLICITY_FACTOR_CTFENCODERS=[...]
			MULTIPLICITY_FACTOR_REST=[...] GEN_TOPO_WIPE_CACHE=[0/1] BEAMTYPE=[PbPb/pp/pPb/cosmic/technical]
			NHBPERTF=[...] GEN_TOPO_ONTHEFLY=1 [Extra environment variables]
			/home/epn/pdp/gen_topo.sh

			R3C-710:
			`pdp_o2pdpsuite_version` is a new field. Its content should be sent in the string as `OVERRIDE_PDPSUITE_VERSION=[...]`.
				In case it is set to `default`, instead of the string `default` the preconfigured default version in consul should be sent.
			`pdp_qcjson_version`: similar to avove, new field. please send as `SET_QCJSON_VERSION`.
				If set to the string `default`, please sent the default version configured in consul instead.
			`pdp_o2_data_processing_hash`: if set to the string `default`, sent the default hash configured in consul instead.
			`odc_n_epns_max_fail` : new field. Please send as `RECO_MAX_FAIL_NODES_OVERRIDE=[...]`.
			`epn_store_raw_data_fraction` new field, please send as `DD_DISK_FRACTION=[...]`.
			`pdp_nr_compute_nodes` removed this field since no longer needed.
				Please send the value of `odc_n_epns` directly as `RECO_NUM_NODES_OVERRIDE=[...]`.
			`pdp_epn_shmid`: new field, please send as `SHM_MANAGER_SHMID=[...]`
			`pdp_epn_shm_recreate`: new field, please send as `SHM_MANAGER_SHM_RECREATE=[0|1]`
		*/

		var (
			pdpConfigOption, o2DPSource, tfbDDMode                                        string
			pdpLibraryFile, pdpLibWorkflowName                                            string
			pdpDetectorList, pdpDetectorExcludeListQc, pdpDetectorExcludeListCalib        string
			pdpWorkflowParams                                                             string
			pdpRawDecoderMultiFactor, pdpCtfEncoderMultiFactor, pdpRecoProcessMultiFactor string
			pdpWipeWorkflowCache, pdpBeamType, pdpNHbfPerTf                               string
			pdpExtraEnvVars, pdpGeneratorScriptPath                                       string
			odcNEpns                                                                      string
			ok                                                                            bool
			accumulator                                                                   []string
			pdpO2PdpSuiteVersion, pdpQcJsonVersion                                        string
			odcNEpnsMaxFail, epnStoreRawDataFraction                                      string
			pdpEpnShmId                                                                   string
			runType                                                                       string
		)
		accumulator = make([]string, 0)

		pdpConfigOption, ok = varStack["pdp_config_option"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP workflow configuration mode")
			return
		}

		switch pdpConfigOption {
		case "Repository hash":
			o2DPSource, ok = varStack["pdp_o2_data_processing_hash"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNWorkflowScript").
					Error("cannot acquire PDP Repository hash")
				return
			}
			if strings.TrimSpace(o2DPSource) == "default" { // if UI sends 'default', we look in Consul
				o2DPSource, ok = configStack["pdp_o2_data_processing_hash"]
				if !ok {
					log.WithField("partition", envId).
						WithField("call", "GenerateEPNWorkflowScript").
						Error("cannot acquire PDP Repository hash default")
					return
				}
			}
			accumulator = append(accumulator, "GEN_TOPO_HASH=1")

		case "Repository path":
			o2DPSource, ok = varStack["pdp_o2_data_processing_path"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNWorkflowScript").
					Error("cannot acquire PDP Repository path")
				return
			}
			accumulator = append(accumulator, "GEN_TOPO_HASH=0")

		case "Manual XML":
			fallthrough
		default:
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("GEN_TOPO_SOURCE='%s'", strings.TrimSpace(o2DPSource)))

		tfbDDMode, ok = varStack["tfb_dd_mode"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire TF Builder mode")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("DDMODE='%s'", strings.TrimSpace(tfbDDMode)))

		pdpLibraryFile, ok = varStack["pdp_topology_description_library_file"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire topology description library file")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("GEN_TOPO_LIBRARY_FILE='%s'", strings.TrimSpace(pdpLibraryFile)))

		pdpLibWorkflowName, ok = varStack["pdp_workflow_name"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP workflow name in topology library file")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("GEN_TOPO_WORKFLOW_NAME='%s'", strings.TrimSpace(pdpLibWorkflowName)))

		pdpDetectorList, ok = varStack["pdp_detector_list_global"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP workflow name in topology library file")
			return
		}
		if strings.TrimSpace(pdpDetectorList) == "default" {
			pdpDetectorList, ok = varStack["detectors"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNWorkflowScript").
					Error("cannot acquire general detector list from varStack")
				return
			}
			detectorsSlice, err := p.parseDetectors(pdpDetectorList)
			if err != nil {
				log.WithField("partition", envId).
					WithField("detectorList", pdpDetectorList).
					WithField("call", "GenerateEPNWorkflowScript").
					Error("cannot parse general detector list")
				return
			}

			// Special case: if the detector list is "default" and ctp_readout_enabled==true, we include TRG
			ctpReadoutEnabled := "false"
			ctpReadoutEnabled, ok = varStack["ctp_readout_enabled"]
			if ok && strings.ToLower(strings.TrimSpace(ctpReadoutEnabled)) == "true" {
				detectorsSlice = append(detectorsSlice, "TRG")
			}
			pdpDetectorList = strings.Join(detectorsSlice, ",")
		}
		accumulator = append(accumulator, fmt.Sprintf("WORKFLOW_DETECTORS='%s'", strings.TrimSpace(pdpDetectorList)))

		pdpDetectorExcludeListQc, ok = varStack["pdp_detector_exclude_list_qc"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire QC detector exclude list in topology library file")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("WORKFLOW_DETECTORS_EXCLUDE_QC='%s'", strings.TrimSpace(pdpDetectorExcludeListQc)))

		pdpDetectorExcludeListCalib, ok = varStack["pdp_detector_exclude_list_calib"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire calibration detector exclude list in topology library file")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("WORKFLOW_DETECTORS_EXCLUDE_CALIB='%s'", strings.TrimSpace(pdpDetectorExcludeListCalib)))

		pdpWorkflowParams, ok = varStack["pdp_workflow_parameters"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP workflow parameters")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("WORKFLOW_PARAMETERS='%s'", strings.TrimSpace(pdpWorkflowParams)))

		odcNEpns, ok = varStack["odc_n_epns"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire ODC number of EPNs")
			return
		}
		odcNEpnsI, err := strconv.Atoi(odcNEpns)
		if err != nil {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot parse ODC number of EPNs")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("RECO_NUM_NODES_OVERRIDE=%d", odcNEpnsI))

		odcNEpnsMaxFail, ok = varStack["odc_n_epns_max_fail"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire ODC number of EPNs max fail")
			return
		}
		odcNEpnsMaxFailI, err := strconv.Atoi(odcNEpnsMaxFail)
		if err != nil {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot parse ODC number of EPNs max fail")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("RECO_MAX_FAIL_NODES_OVERRIDE=%d", odcNEpnsMaxFailI))

		pdpRawDecoderMultiFactor, ok = varStack["pdp_raw_decoder_multi_factor"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP number of raw decoder processing instances")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("MULTIPLICITY_FACTOR_RAWDECODERS=%s", strings.TrimSpace(pdpRawDecoderMultiFactor)))

		pdpCtfEncoderMultiFactor, ok = varStack["pdp_ctf_encoder_multi_factor"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP number of CTF encoder processing instances")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("MULTIPLICITY_FACTOR_CTFENCODERS=%s", strings.TrimSpace(pdpCtfEncoderMultiFactor)))

		pdpRecoProcessMultiFactor, ok = varStack["pdp_reco_process_multi_factor"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP number of other reconstruction processing instances")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("MULTIPLICITY_FACTOR_REST=%s", strings.TrimSpace(pdpRecoProcessMultiFactor)))

		pdpBeamType, ok = varStack["pdp_beam_type"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire beam type")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("BEAMTYPE='%s'", strings.TrimSpace(pdpBeamType)))

		pdpNHbfPerTf, ok = varStack["pdp_n_hbf_per_tf"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire number of HBFs per TF")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("NHBPERTF=%s", strings.TrimSpace(pdpNHbfPerTf)))

		accumulator = append(accumulator, "GEN_TOPO_ONTHEFLY=1")

		pdpO2PdpSuiteVersion, ok = varStack["pdp_o2pdpsuite_version"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP Suite version")
			return
		}
		if strings.TrimSpace(pdpO2PdpSuiteVersion) == "default" { // if UI sends 'default', we look in Consul
			pdpO2PdpSuiteVersion, ok = configStack["pdp_o2pdpsuite_version"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNWorkflowScript").
					Error("cannot acquire PDP Suite version default")
				return
			}
		}
		accumulator = append(accumulator, fmt.Sprintf("OVERRIDE_PDPSUITE_VERSION='%s'", pdpO2PdpSuiteVersion))

		// SET_QCJSON_VERSION does not come from user input or vars any more, it's instead a direct query to QC runtime
		pdpQcJsonVersion, err = apricot.Instance().GetRuntimeEntry("qc", "config_hash")
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP QCJson config_hash from QC runtime KV")
			return
		}

		accumulator = append(accumulator, fmt.Sprintf("SET_QCJSON_VERSION='%s'", pdpQcJsonVersion))

		epnStoreRawDataFraction, ok = varStack["epn_store_raw_data_fraction"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire EPN DD disk raw data fraction")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("DD_DISK_FRACTION='%s'", epnStoreRawDataFraction))

		pdpEpnShmId, ok = varStack["pdp_epn_shmid"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP EPN SHMID")
			return
		}
		accumulator = append(accumulator, fmt.Sprintf("SHM_MANAGER_SHMID='%s'", pdpEpnShmId))

		runType, ok = varStack["run_type"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Warn("could not get get variable run_type from environment context, using NONE")
			runType = "NONE"
		}
		accumulator = append(accumulator, fmt.Sprintf("RUNTYPE=%s", strings.TrimSpace(runType)))

		pdpExtraEnvVars, ok = varStack["pdp_extra_env_vars"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP extra environment variables")
			return
		}
		accumulator = append(accumulator, strings.TrimSpace(pdpExtraEnvVars))

		pdpGeneratorScriptPath, ok = varStack["pdp_generator_script_path"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot acquire PDP generator script path")
			return
		}
		accumulator = append(accumulator, strings.TrimSpace(pdpGeneratorScriptPath))

		out = strings.Join(accumulator, " ")

		// before we ship out the payload, we take the hash of the full string and prepend a few last variables with the
		// hash of everything else that follows, except ECS_ENVIRONMENT_ID and GEN_TOPO_WIPE_CACHE, the only
		// variables that must stay unhashed
		// see https://alice.its.cern.ch/jira/browse/OCTRL-736
		hash := md5.Sum([]byte(out))
		hashS := hex.EncodeToString(hash[:])
		out = fmt.Sprintf("GEN_TOPO_CACHE_HASH=%s", hashS) + " " + out

		pdpWipeWorkflowCache, ok = varStack["pdp_wipe_workflow_cache"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Warn("cannot acquire PDP workflow cache wipe option, assuming false")
			pdpWipeWorkflowCache = "false"
		}
		pdpWipeWorkflowCacheB, err := strconv.ParseBool(pdpWipeWorkflowCache)
		if err != nil {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNWorkflowScript").
				Error("cannot parse PDP workflow cache wipe option")
			pdpWipeWorkflowCacheB = false
		}
		pdpWipeWorkflowCacheI := 0
		if pdpWipeWorkflowCacheB {
			pdpWipeWorkflowCacheI = 1
		}
		accumulator = append(accumulator, fmt.Sprintf("GEN_TOPO_WIPE_CACHE=%d", pdpWipeWorkflowCacheI))

		// finally we prepend ECS_ENVIRONMENT_ID
		out = fmt.Sprintf("ECS_ENVIRONMENT_ID=%s", envId) + " " + out

		return
	}
	stack["GenerateEPNTopologyFullname"] = func() (out string) {
		pdpConfigOption, ok := varStack["pdp_config_option"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNTopologyFullname").
				Error("cannot acquire PDP workflow configuration mode")
			return
		}

		pdpLibraryFile, ok := varStack["pdp_topology_description_library_file"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNTopologyFullname").
				Error("cannot acquire topology description library file")
			return
		}

		pdpLibWorkflowName, ok := varStack["pdp_workflow_name"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "GenerateEPNTopologyFullname").
				Error("cannot acquire PDP workflow name in topology library file")
			return
		}

		odcTopologyFullname := ""
		switch pdpConfigOption {
		case "Repository hash":
			o2DPSource, ok := varStack["pdp_o2_data_processing_hash"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNTopologyFullname").
					Error("cannot acquire PDP Repository hash")
				return
			}
			odcTopologyFullname = "(hash, " + o2DPSource + ", " + pdpLibraryFile + ", " + pdpLibWorkflowName + ")"

		case "Repository path":
			o2DPSource, ok := varStack["pdp_o2_data_processing_path"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNTopologyFullname").
					Error("cannot acquire PDP Repository hash")
				return
			}
			odcTopologyFullname = "(path, " + o2DPSource + ", " + pdpLibraryFile + ", " + pdpLibWorkflowName + ")"

		case "Manual XML":
			odc_topology, ok := varStack["odc_topology"]
			if !ok {
				log.WithField("partition", envId).
					WithField("call", "GenerateEPNTopologyFullname").
					Error("cannot acquire ODC topology variable")
				return
			}
			odcTopologyFullname = "(xml, " + odc_topology + ")"
		}
		return odcTopologyFullname
	}
	return stack
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("CallStack cannot acquire environment ID")
		return
	}

	paddingTimeout := ODC_PADDING_TIMEOUT
	paddingTimeoutStr, ok := varStack["odc_padding_timeout"]
	if ok {
		var err error
		paddingTimeout, err = time.ParseDuration(paddingTimeoutStr)
		if err != nil {
			paddingTimeout = ODC_PADDING_TIMEOUT
			log.Debugf("CallStack cannot acquire ODC padding timeout, defaulting to %s", ODC_PADDING_TIMEOUT.String())
		}
	} else {
		log.Debugf("CallStack cannot acquire ODC padding timeout, defaulting to %s", ODC_PADDING_TIMEOUT.String())
	}

	stack = make(map[string]interface{})
	stack["PartitionInitialize"] = func() (out string) {
		// ODC Run
		var err error = nil

		log.WithField("partition", envId).Debugf("preparing call odc.PartitionInitialize")

		var (
			pdpConfigOption, script, topology, plugin, resources string
		)
		ok := false
		isManualXml := false
		callFailedStr := "EPN PartitionInitialize call failed"

		pdpConfigOption, ok = varStack["pdp_config_option"]
		if !ok {
			msg := "cannot acquire PDP workflow configuration mode"
			log.WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}
		switch pdpConfigOption {
		case "Repository hash":
			fallthrough
		case "Repository path":
			script, ok = varStack["odc_script"]
			if !ok {
				msg := "cannot acquire ODC script, make sure GenerateEPNWorkflowScript is called and its " +
					"output is written to odc_script"
				log.WithField("partition", envId).
					WithField("call", "PartitionInitialize").
					Error(msg)
				call.VarStack["__call_error_reason"] = msg
				call.VarStack["__call_error"] = callFailedStr
				return
			}

		case "Manual XML":
			topology, ok = varStack["odc_topology"]
			if !ok {
				msg := "cannot acquire ODC topology"
				log.WithField("partition", envId).
					WithField("call", "PartitionInitialize").
					Error(msg)
				call.VarStack["__call_error_reason"] = msg
				call.VarStack["__call_error"] = callFailedStr
				return
			}
			isManualXml = true

		default:
			msg := "cannot acquire valid PDP workflow configuration mode value"
			log.WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				WithField("value", pdpConfigOption).
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		plugin, ok = varStack["odc_plugin"]
		if !ok {
			msg := "cannot acquire ODC RMS plugin declaration"
			log.WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		resources, ok = varStack["odc_resources"]
		if !ok {
			msg := "cannot acquire ODC resources declaration"
			log.WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		timeout := callable.AcquireTimeout(ODC_PARTITIONINITIALIZE_TIMEOUT, varStack, "PartitionInitialize", envId)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err = handleRun(ctx, p.odcClient, isManualXml, map[string]string{
			"topology":  topology,
			"script":    script,
			"plugin":    plugin,
			"resources": resources,
		},
			paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PartitionInitialize").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
			return
		}
		log.WithField("partition", envId).Debugf("finished call odc.PartitionInitialize with SUCCESS")
		return
	}
	stack["Configure"] = func() (out string) {
		// ODC SetProperties + Configure

		callFailedStr := "EPN Configure call failed"

		timeout := callable.AcquireTimeout(ODC_CONFIGURE_TIMEOUT, varStack, "Configure", envId)

		arguments := make(map[string]string)
		arguments["environment_id"] = envId

		runType, ok := varStack["run_type"]
		if ok {
			arguments["run_type"] = runType
		} else {
			log.WithField("partition", envId).
				WithField("call", "Configure").
				Warn("could not get get variable run_type from environment context")
		}

		lhcPeriod, ok := varStack["lhc_period"]
		if ok {
			arguments["lhc_period"] = lhcPeriod
		} else {
			log.WithField("partition", envId).
				WithField("call", "Configure").
				Warn("could not get get variable lhc_period from environment context")
		}

		forceRunAsRaw, ok := varStack["pdp_force_run_as_raw"]
		if ok {
			arguments["force_run_as_raw"] = forceRunAsRaw
		} else {
			log.WithField("partition", envId).
				WithField("call", "Configure").
				Warn("could not get get variable force_run_as_raw from environment context")
		}

		detectorListS, ok := varStack["detectors"]
		if ok {
			detectorsSlice, err := p.parseDetectors(detectorListS)
			if err == nil {
				arguments["detectors"] = strings.Join(detectorsSlice, ",")
			} else {
				log.WithField("partition", envId).
					WithField("detectorList", detectorsSlice).
					WithField("call", "Configure").
					Warn("cannot parse general detector list")
			}
		} else {
			log.WithField("partition", envId).
				WithField("call", "Configure").
				Warn("cannot acquire general detector list from varStack")
		}

		// Push orbit-reset-time if pdp_override_run_start_time set
		pdpOverrideRunStartTime, ok := varStack["pdp_override_run_start_time"]
		if ok && len(pdpOverrideRunStartTime) > 0 {
			arguments["orbit-reset-time"] = pdpOverrideRunStartTime
			if strings.Contains(runType, "SYNTHETIC") {
				log.WithField("partition", envId).
					WithField("call", "Configure").
					WithField("runType", runType).
					Infof("overriding run start time (orbit-reset-time) to %s for SYNTHETIC run", pdpOverrideRunStartTime)
			} else {
				log.WithField("partition", envId).
					WithField("call", "Configure").
					WithField("runType", runType).
					Warnf("overriding run start time (orbit-reset-time) to %s for non-SYNTHETIC run", pdpOverrideRunStartTime)
			}
		} else { // no run start override defined
			if strings.Contains(runType, "SYNTHETIC") {
				log.WithField("partition", envId).
					WithField("call", "Configure").
					WithField("runType", runType).
					Warnf("requested SYNTHETIC run but run start time (orbit-reset-time) override not provided")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleConfigure(ctx, p.odcClient, arguments, paddingTimeout, envId)
		if err != nil {
			log.WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "Configure").
				WithError(err).
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}

		return
	}
	stack["Reset"] = func() (out string) {
		// ODC Reset

		timeout := callable.AcquireTimeout(ODC_RESET_TIMEOUT, varStack, "Reset", envId)

		callFailedStr := "EPN Reset call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleReset(ctx, p.odcClient, nil, paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "Reset").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["PartitionTerminate"] = func() (out string) {
		// ODC Terminate + Shutdown

		timeout := callable.AcquireTimeout(ODC_PARTITIONTERMINATE_TIMEOUT, varStack, "PartitionTerminate", envId)

		callFailedStr := "EPN PartitionTerminate call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handlePartitionTerminate(ctx, p.odcClient, nil, paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PartitionTerminate").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["Start"] = func() (out string) { // must formally return string even when we return nothing

		// ODC SetProperties + Start

		rn, ok := varStack["run_number"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "Start").
				Warn("cannot acquire run number for ODC")
		}
		cleanupCountS, ok := varStack["__fmq_cleanup_count"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "Start").
				Warn("cannot acquire FairMQ devices cleanup count for ODC")
		}

		var (
			runNumberu64 uint64
			cleanupCount int
			err          error
		)
		callFailedStr := "EPN Start call failed"

		runNumberu64, err = strconv.ParseUint(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for ODC SOR")
			runNumberu64 = 0
		}
		cleanupCount, err = strconv.Atoi(cleanupCountS)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire FairMQ devices cleanup count for ODC SOR")
			cleanupCount = 1
		}

		timeout := callable.AcquireTimeout(ODC_START_TIMEOUT, varStack, "Start", envId)

		arguments := make(map[string]string)
		arguments["run_number"] = rn
		arguments["runNumber"] = rn
		arguments["cleanup"] = strconv.Itoa(cleanupCount)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = handleStart(ctx, p.odcClient, arguments, paddingTimeout, envId, runNumberu64)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "Start").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["Stop"] = func() (out string) {
		// ODC Stop

		rn, ok := varStack["run_number"]
		if !ok {
			log.WithField("partition", envId).
				WithField("call", "Start").
				Warn("cannot acquire run number for ODC")
		}
		var (
			runNumberu64 uint64
			err          error
		)
		callFailedStr := "EPN Stop call failed"

		runNumberu64, err = strconv.ParseUint(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for DCS SOR")
			runNumberu64 = 0
		}

		timeout := callable.AcquireTimeout(ODC_STOP_TIMEOUT, varStack, "Stop", envId)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = handleStop(ctx, p.odcClient, nil, paddingTimeout, envId, runNumberu64)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "Stop").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["EnsureCleanup"] = func() (out string) {
		// ODC Shutdown for current env + all orphans

		timeout := callable.AcquireTimeout(ODC_GENERAL_OP_TIMEOUT, varStack, "EnsureCleanup", envId)

		callFailedStr := "EPN EnsureCleanup call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleCleanup(ctx, p.odcClient, nil, paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "EnsureCleanup").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["PreDeploymentCleanup"] = func() (out string) {
		// ODC Shutdown for all orphans

		timeout := callable.AcquireTimeout(ODC_GENERAL_OP_TIMEOUT, varStack, "PreDeploymentCleanup", envId)

		callFailedStr := "EPN PreDeploymentCleanup call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleCleanup(ctx, p.odcClient, nil, paddingTimeout, "")
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "PreDeploymentCleanup").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["ConfigureLegacy"] = func() (out string) {
		// ODC Run + SetProperties + Configure

		var (
			pdpConfigOption, script, topology, plugin, resources string
		)
		ok := false
		isManualXml := false
		callFailedStr := "EPN ConfigureLegacy call failed"

		pdpConfigOption, ok = varStack["pdp_config_option"]
		if !ok {
			msg := "cannot acquire PDP workflow configuration mode"
			log.WithField("partition", envId).
				WithField("call", "ConfigureLegacy").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}
		switch pdpConfigOption {
		case "Repository hash":
			fallthrough
		case "Repository path":
			script, ok = varStack["odc_script"]
			if !ok {
				msg := "cannot acquire ODC script, make sure GenerateEPNWorkflowScript is called and its " +
					"output is written to odc_script"
				log.WithField("partition", envId).
					WithField("call", "ConfigureLegacy").
					Error(msg)
				call.VarStack["__call_error_reason"] = msg
				call.VarStack["__call_error"] = callFailedStr
				return
			}

		case "Manual XML":
			topology, ok = varStack["odc_topology"]
			if !ok {
				msg := "cannot acquire ODC topology"
				log.WithField("partition", envId).
					WithField("call", "ConfigureLegacy").
					Error(msg)
				call.VarStack["__call_error_reason"] = msg
				call.VarStack["__call_error"] = callFailedStr
				return
			}
			isManualXml = true

		default:
			msg := "cannot acquire valid PDP workflow configuration mode value"
			log.WithField("partition", envId).
				WithField("call", "ConfigureLegacy").
				WithField("value", pdpConfigOption).
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		plugin, ok = varStack["odc_plugin"]
		if !ok {
			msg := "cannot acquire ODC RMS plugin declaration"
			log.WithField("partition", envId).
				WithField("call", "ConfigureLegacy").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		resources, ok = varStack["odc_resources"]
		if !ok {
			msg := "cannot acquire ODC resources declaration"
			log.WithField("partition", envId).
				WithField("call", "ConfigureLegacy").
				Error(msg)
			call.VarStack["__call_error_reason"] = msg
			call.VarStack["__call_error"] = callFailedStr
			return
		}

		timeout := callable.AcquireTimeout(ODC_CONFIGURE_TIMEOUT, varStack, "Configure", envId)

		arguments := make(map[string]string)
		arguments["environment_id"] = envId

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleConfigureLegacy(ctx, p.odcClient, arguments, isManualXml, topology, script, plugin, resources, paddingTimeout, envId)
		if err != nil {
			log.WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "ConfigureLegacy").
				WithError(err).Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}

		return
	}
	stack["ResetLegacy"] = func() (out string) {
		// ODC Reset + Terminate + Shutdown

		timeout := callable.AcquireTimeout(ODC_RESET_TIMEOUT, varStack, "Reset", envId)

		callFailedStr := "EPN ResetLegacy call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleResetLegacy(ctx, p.odcClient, nil, paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "ResetLegacy").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["EnsureCleanupLegacy"] = func() (out string) {
		// ODC Reset + Terminate + Shutdown for current env

		timeout := callable.AcquireTimeout(ODC_GENERAL_OP_TIMEOUT, varStack, "EnsureCleanupLegacy", envId)

		callFailedStr := "EPN EnsureCleanupLegacy call failed"

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := handleCleanupLegacy(ctx, p.odcClient, nil, paddingTimeout, envId)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "EnsureCleanupLegacy").
				Error("ODC error")
			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	return
}

func (p *Plugin) parseDetectors(detectorsParam string) (detectors []string, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(detectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).
			Error("error processing EPN/PDP detectors list")
		return
	}
	detectors = detectorsSlice
	return
}

func (p *Plugin) Destroy() error {
	p.cachedStatusCancelFunc()
	return p.odcClient.Close()
}
