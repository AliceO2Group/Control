/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2022 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <piotr.jan.konopka@cern.ch>
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

package ccdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/types"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/runtype"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/integration"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "ccdbclient")

type GeneralRunParameters struct {
	runNumber                  uint32
	runType                    runtype.RunType
	startTimeMs                string // we keep it as string, to avoid converting back and forth from time.Time
	endTimeMs                  string // we keep it as string, to avoid converting back and forth from time.Time
	detectors                  []string
	continuousReadoutDetectors []string
	triggeringDetectors        []string
	hbfPerTf                   uint32 // number of HeartBeatFrames per TimeFrame
	lhcPeriod                  string
}

func parseDetectors(detectorsParam string) (detectors []string, err error) {
	detectorsSlice := make([]string, 0)
	bytes := []byte(detectorsParam)
	err = json.Unmarshal(bytes, &detectorsSlice)
	if err != nil {
		log.WithError(err).
			Error("error processing the detectors list")
		return
	}
	return detectorsSlice, nil
}

func NewGRPObject(varStack map[string]string) *GeneralRunParameters {

	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("cannot acquire environment ID")
		return nil
	}

	runNumberStr, ok := varStack["run_number"]
	if !ok {
		log.WithField("partition", envId).
			Error("cannot acquire run number for GRP object")
		return nil
	}
	runNumber, err := strconv.ParseUint(runNumberStr, 10, 32)
	if err != nil {
		log.WithError(err).
			WithField("partition", envId).
			Errorf("cannot convert run number '%s' to an integer", runNumberStr)
		return nil
	}

	runType := runtype.NONE
	runTypeStr, ok := varStack["run_type"]
	if ok {
		intRt, err := runtype.RunTypeString(runTypeStr)
		if err == nil {
			runType = intRt
		}
	}

	startTime := varStack["run_start_time_ms"]
	endTime := varStack["run_end_time_ms"]

	// use the fake run start time if available
	pdpOverrideRunStartTime, ok := varStack["pdp_override_run_start_time"]
	if ok && len(pdpOverrideRunStartTime) > 0 {
		if strings.Contains(runTypeStr, "SYNTHETIC") {
			log.WithField("partition", envId).
				WithField("runType", runTypeStr).
				Infof("overriding run start time in the GRP object to %s for SYNTHETIC run", pdpOverrideRunStartTime)
		} else {
			log.WithField("partition", envId).
				WithField("runType", runTypeStr).
				Warnf("overriding run start time to %s for non-SYNTHETIC run", pdpOverrideRunStartTime)
		}
		if len(endTime) > 0 {
			// calculate eor time as pdp_override_run_start_time + real run duration
			startTimeNumber, err := strconv.ParseUint(startTime, 10, 64)
			if err != nil {
				log.WithError(err).
					WithField("partition", envId).
					Errorf("could not parse run startTime: %s", startTime)
			}
			pdpOverrideRunStartTimeNumber, err := strconv.ParseUint(pdpOverrideRunStartTime, 10, 64)
			if err != nil {
				log.WithError(err).
					WithField("partition", envId).
					Errorf("could not parse pdpOverrideRunStartTimeNumber: %s", pdpOverrideRunStartTime)
			}
			endTimeNumber, err := strconv.ParseUint(endTime, 10, 64)
			if err != nil {
				log.WithError(err).
					WithField("partition", envId).
					Errorf("could not parse run endTime: %s", endTime)
			}
			if endTimeNumber <= startTimeNumber {
				log.WithError(err).
					WithField("partition", envId).
					Errorf("endTimeNumber (%d) is smaller or equal to startTimeNumber (%d)", endTimeNumber, startTimeNumber)
			}

			runDuration := endTimeNumber - startTimeNumber
			endTime = strconv.FormatUint(pdpOverrideRunStartTimeNumber+runDuration, 10)
		}
		startTime = pdpOverrideRunStartTime
	} else if strings.Contains(runTypeStr, "SYNTHETIC") {
		log.WithField("partition", envId).
			WithField("runType", runTypeStr).
			Warnf("requested SYNTHETIC run but run start time override not provided")
	}

	detectorsStr, ok := varStack["detectors"]
	if !ok {
		log.WithField("partition", envId).
			Error("cannot acquire general detector list from varStack")
	}
	detectorsSlice, err := parseDetectors(detectorsStr)
	if err != nil {
		log.WithField("partition", envId).
			Error("cannot parse general detector list")
		return nil
	}

	// Special case: if ctp_readout_enabled, we must add TRG to the detectors list
	ctpReadoutEnabled := "false"
	ctpReadoutEnabled, ok = varStack["ctp_readout_enabled"]
	if ok && strings.ToLower(strings.TrimSpace(ctpReadoutEnabled)) == "true" {
		detectorsSlice = append(detectorsSlice, "TRG")
	}

	// TODO fill once we have those available
	var continuousReadoutDetectors []string
	var triggeringDetectors []string

	hbfPerTf, err := strconv.ParseUint(varStack["n_hbf_per_tf"], 10, 32)
	if err != nil {
		log.WithError(err).
			WithField("partition", envId).
			Error("cannot acquire run number for Run Start")
		return nil
	}

	lhcPeriod, ok := varStack["lhc_period"]
	if !ok {
		log.WithField("partition", envId).
			WithField("runNumber", runNumber).
			Debug("CCDB interface could not retrieve 'lhc_period', putting 'Unknown'.")
		lhcPeriod = "Unknown"
	}

	return &GeneralRunParameters{
		uint32(runNumber),
		runType,
		startTime,
		endTime,
		detectorsSlice,
		continuousReadoutDetectors,
		triggeringDetectors,
		uint32(hbfPerTf),
		lhcPeriod,
	}
}

type Plugin struct {
	ccdbUrl      string
	existingRuns map[uint32]types.Nil // using map, because it is more convenient to add, find, delete elements than slice
}

func NewPlugin(endpoint string) integration.Plugin {
	_, err := url.Parse(endpoint)
	if err != nil {
		log.WithField("endpoint", endpoint).
			WithError(err).
			Error("bad service endpoint")
		return nil
	}

	return &Plugin{
		ccdbUrl:      endpoint,
		existingRuns: make(map[uint32]types.Nil),
	}
}

func (p *Plugin) GetName() string {
	return "ccdb"
}

func (p *Plugin) GetPrettyName() string {
	return "CCDB"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("ccdbEndpoint")
}

func (p *Plugin) GetConnectionState() string {
	return "READY"
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	return ""
}

func (p *Plugin) Init(instanceId string) error {
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
	return stack
}

func (p *Plugin) NewCcdbGrpWriteCommand(grp *GeneralRunParameters, ccdbUrl string, refresh bool) (cmd string, err error) {
	// o2-ecs-grp-create -h
	//Create GRP-ECS object and upload to the CCDB
	//Usage:
	//  -h [ --help ]                         Print this help message
	//  -p [ --period ] arg                   data taking period
	//  -r [ --run ] arg                      run number
	//  -t [ --run-type ] arg (=0)            run type
	//  -n [ --hbf-per-tf ] arg (=128)        number of HBFs per TF
	//  -d [ --detectors ] arg (=all)         comma separated list of detectors
	//  -c [ --continuous ] arg (=ITS,TPC,TOF,MFT,MCH,MID,ZDC,FT0,FV0,FDD,CTP)
	//                                        comma separated list of detectors in
	//                                        continuous readout mode
	//  -g [ --triggering ] arg (=FT0,FV0)    comma separated list of detectors
	//                                        providing a trigger
	//  -s [ --start-time ] arg (=0)          run start time in ms, now() if 0
	//  -e [ --end-time ] arg (=0)            run end time in ms, start-time+3days is
	//                                        used if 0
	//  --ccdb-server arg (=http://alice-ccdb.cern.ch)
	//                                        CCDB server for upload, local file if
	//                                        empty
	// --refresh                              refresh server cache after upload

	cmd = "source /etc/profile.d/o2.sh && o2-ecs-grp-create"
	if len(grp.lhcPeriod) == 0 {
		return "", fmt.Errorf("could not create a command for CCDB interface because LHC period is empty")
	}
	cmd += " -p " + grp.lhcPeriod
	if grp.runNumber == 0 {
		return "", fmt.Errorf("could not create a command for CCDB interface because run number is 0")
	}
	cmd += " -r " + strconv.FormatUint(uint64(grp.runNumber), 10)
	if refresh {
		cmd += " --refresh"
	}
	if grp.hbfPerTf != 0 {
		cmd += " -n " + strconv.FormatUint(uint64(grp.hbfPerTf), 10)
	}
	if grp.runType != runtype.NONE {
		cmd += " -t " + strconv.Itoa(int(grp.runType))
	}
	if len(grp.detectors) != 0 {
		cmd += " -d \"" + strings.Join(grp.detectors, ",") + "\""
	}
	if len(grp.continuousReadoutDetectors) != 0 {
		cmd += " -c \"" + strings.Join(grp.continuousReadoutDetectors, ",") + "\""
	}
	if len(grp.triggeringDetectors) != 0 {
		cmd += " -g \"" + strings.Join(grp.triggeringDetectors, ",") + "\""
	}
	if len(grp.startTimeMs) > 0 {
		cmd += " -s " + grp.startTimeMs
	}
	if len(grp.endTimeMs) > 0 {
		cmd += " -e " + grp.endTimeMs
	}

	cmd += " --ccdb-server " + ccdbUrl
	return
}

func (p *Plugin) CallStack(data interface{}) (stack map[string]interface{}) {
	call, ok := data.(*callable.Call)
	if !ok {
		return
	}
	varStack := call.VarStack
	envId, ok := varStack["environment_id"]
	if !ok {
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	stack["RunStart"] = func() (out string) { // must formally return string even when we return nothing
		log.WithField("call", "RunStart").
			WithField("partition", envId).Debug("performing CCDB interface Run Start")

		grp := NewGRPObject(varStack)
		if grp == nil {
			return
		}
		p.existingRuns[grp.runNumber] = types.Nil{}
		err := p.uploadCurrentGRP(grp, envId, true)
		if err != nil {
			log.WithField("call", "RunStop").
				WithField("partition", envId).Error(err.Error())
		}
		return
	}
	stack["RunStop"] = func() (out string) {
		log.WithField("call", "RunStop").
			WithField("partition", envId).Debug("checking if a CCDB End Of Run GRP should be published")

		grp := NewGRPObject(varStack)
		if grp == nil {
			log.WithField("call", "RunStop").
				WithField("partition", envId).
				Debug("probably went to ERROR while not in RUNNING, doing nothing")
			return
		}
		_, runExists := p.existingRuns[grp.runNumber]
		if runExists {
			delete(p.existingRuns, grp.runNumber)
			err := p.uploadCurrentGRP(grp, envId, false)
			if err != nil {
				log.WithField("call", "RunStop").
					WithField("partition", envId).Error(err.Error())
			}
		} else {
			log.WithField("call", "RunStop").
				WithField("partition", envId).
				Debugf("most likely a GRP EOR object for run %d already has been published, doing nothing", grp.runNumber)
		}
		return
	}
	return
}

func (p *Plugin) uploadCurrentGRP(grp *GeneralRunParameters, envId string, refresh bool) error {

	if grp == nil {
		return errors.New(fmt.Sprintf("Failed to create a GRP object"))
	}
	log.WithField("partition", envId).Debug(
		fmt.Sprintf("GRP: %d, %s, %s, %s, %d, %s, %s, %s, %s",
			grp.runNumber, grp.runType.String(), grp.startTimeMs, grp.endTimeMs, grp.hbfPerTf, grp.lhcPeriod,
			strings.Join(grp.detectors, ","), strings.Join(grp.triggeringDetectors, ","), strings.Join(grp.continuousReadoutDetectors, ",")))
	cmdStr, err := p.NewCcdbGrpWriteCommand(grp, p.ccdbUrl, refresh)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to build a GRP to CCDB upload command: " + err.Error()))
	}
	log.WithField("partition", envId).Debug(fmt.Sprintf("CCDB GRP upload command: '" + cmdStr + "'"))

	const timeoutSeconds = 10
	ctx, cancel := context.WithTimeout(context.Background(), timeoutSeconds*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	// execute the DPL command in the repo of the workflow used
	cmd.Dir = "/tmp"
	cmdOut, err := cmd.CombinedOutput()
	log.Debug("CCDB GRP upload command out: " + string(cmdOut))
	if ctx.Err() == context.DeadlineExceeded {
		return errors.New(fmt.Sprintf("The command to upload GRP to CCDB timed out (" + strconv.Itoa(timeoutSeconds) + "s)."))
	}
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to run the command to upload GRP to CCDB: " + err.Error() + "\ncommand out : " + string(cmdOut)))
	}
	return nil
}

func (p *Plugin) Destroy() error {
	return nil
}
