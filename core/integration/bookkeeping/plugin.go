/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.guyot@cern.ch>
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

//go:generate protoc --go_out=. --go-grpc_out=require_unimplemented_servers=false:. protos/bookkeeping.proto

package bookkeeping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	clientAPI "github.com/AliceO2Group/Bookkeeping/go-api-client/src"
	sw "github.com/AliceO2Group/Bookkeeping/go-api-client/src/go-client-generated"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/utils/uid"
	"github.com/AliceO2Group/Control/core/environment"
	"github.com/AliceO2Group/Control/core/integration"
	bkpb "github.com/AliceO2Group/Control/core/integration/bookkeeping/protos"
	"github.com/AliceO2Group/Control/core/workflow/callable"
	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	BK_DIAL_TIMEOUT       = 2 * time.Hour
	BK_GENERAL_OP_TIMEOUT = 45 * time.Second
)

type Plugin struct {
	bookkeepingHost string
	bookkeepingPort int

	bookkeepingClient *RpcClient
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
		bookkeepingHost:   u.Hostname(),
		bookkeepingPort:   portNumber,
		bookkeepingClient: nil,
	}
}

func (p *Plugin) GetName() string {
	return "bookkeeping"
}

func (p *Plugin) GetPrettyName() string {
	return "Bookkeeping"
}

func (p *Plugin) GetEndpoint() string {
	return viper.GetString("bookkeepingBaseUri")
}

func (p *Plugin) GetConnectionState() string {
	if p == nil || p.bookkeepingClient == nil {
		return "UNKNOWN"
	}
	return p.bookkeepingClient.conn.GetState().String()
}

func (p *Plugin) GetData(environmentIds []uid.ID) string {
	if p == nil || p.bookkeepingClient == nil {
		return ""
	}

	partitionStates := make(map[string]string)

	for _, envId := range environmentIds {
		// ADD HERE
	}

	out, err := json.Marshal(partitionStates)
	if err != nil {
		return ""
	}
	return string(out[:])
}

func (p *Plugin) Init(instanceId string) error {
	if p.bookkeepingClient == nil {
		cxt, cancel := context.WithCancel(context.Background())
		p.bookkeepingClient = NewClient(cxt, cancel, viper.GetString("bookkeepingBaseUri"))
		if p.bookkeepingClient == nil {
			return fmt.Errorf("failed to connect to Bookkeeping service on %s", viper.GetString("bookkeepingBaseUri"))
		}
		apiUrl, err := url.Parse(viper.GetString("bookkeepingBaseUri"))
		if err == nil {
			apiUrl.Path = path.Join(apiUrl.Path + "api")
			clientAPI.InitializeApi(apiUrl.String(), viper.GetString("bookkeepingToken"))
		} else {
			log.WithField("error", err.Error()).
				Error("unable to parse the Bookkeeping base URL")
			clientAPI.InitializeApi(path.Join(viper.GetString("bookkeepingBaseUri")+"api"), viper.GetString("bookkeepingToken"))
		}
		log.Debug("Bookkeeping plugin initialized")
	}
	return nil
}

func (p *Plugin) ObjectStack(_ map[string]string) (stack map[string]interface{}) {
	stack = make(map[string]interface{})
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
		log.Error("cannot acquire environment ID")
		return
	}

	stack = make(map[string]interface{})
	stack["StartOfRun"] = func() (out string) { // must formally return string even when we return nothing
		var err error
		callFailedStr := "Bookkeeping StartOfRun call failed"

		rn := varStack["run_number"]
		var runNumber64 int64
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithField("partition", envId).
				WithError(err).
				Error("cannot acquire run number for Bookkeeping SOR")
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping SOR for detectors: %s", strings.Join(detectors.StringSlice(), " "))

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, StartOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		if p.bookkeepingClient.GetConnState() != connectivity.Ready {
			err = fmt.Errorf("Bookkeeping client connection not available, StartOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		envMan := environment.ManagerInstance()
		env := envMan.Environment(envId)
		runNumber := env.currentRunNumber
		runType := env.GetRunType().String()

		args := controlcommands.PropertyMap{
			"runNumber": strconv.FormatUint(uint64(runNumber), 10),
		}

		flps := env.GetFLPs()
		dd_enabled, _ := strconv.ParseBool(env.GetKV("", "dd_enabled"))
		dcs_enabled, _ := strconv.ParseBool(env.GetKV("", "dcs_enabled"))
		epn_enabled, _ := strconv.ParseBool(env.GetKV("", "epn_enabled"))
		odc_topology := env.GetKV("", "odc_topology")
		// GetString of active detectors and pass it to the BK API
		detectors := strings.Join(env.GetActiveDetectors().StringList(), ",")
		var runtypeAPI sw.RunType
		switch runType {
		case string(sw.TECHNICAL_RunType):
			runtypeAPI = sw.TECHNICAL_RunType
		case string(sw.COSMICS_RunType):
			runtypeAPI = sw.COSMICS_RunType
		case string(sw.PHYSICS_RunType):
			runtypeAPI = sw.PHYSICS_RunType
		default:
			// log Runtype is %s and it is not valid overwrite with TECHNICAL_RunType
			runtypeAPI = sw.TECHNICAL_RunType
		}

		timeO2start := time.Now().UnixMilli()
		timeTrgstart := time.Now().UnixMilli()

		in := bkpb.RunCreationRequest{
			EnvironmentId = env.Id().String(),
			NDetectors = len(env.GetActiveDetectors()),
			NEpns = 0,
			NFlps = len(flps),
			RunNumber = int32(runNumber)
			RunType = runtypeAPI,
			TimeO2start = timeO2start,
			TimeTrgStart = timeTrgstart,
			Dd_flp = dd_enabled,
			Dcs = dcs_enabled,
			Epn = epn_enabled,
			EpnTopology = odc_topology,
			Detectors = detectors,
		}

		clientAPI.CreateRun(env.Id().String(), int32(len(env.GetActiveDetectors())), int32(0), int32(len(flps)), int32(runNumber), runtypeAPI, timeO2start, timeTrgstart, dd_enabled, dcs_enabled, epn_enabled, odc_topology, sw.Detectors(detectors))

		log.WithField("runType", runType).
			WithField("partition", env.Id().String()).
			WithField("runNumber", runNumber).
			Debug("CreateRun call done")
/*
		for _, flp := range flps {
			the.BookkeepingAPI().CreateFlp(flp, flp, int32(runNumber))
		}

		the.BookkeepingAPI().CreateLog(env.GetVarsAsString(), fmt.Sprintf("Log for run %s and environment %s", args["runNumber"], env.Id().String()), args["runNumber"], -1)
*/
		var stream bkpb.Configurator_StartOfRunClient
		timeout := callable.AcquireTimeout(BK_GENERAL_OP_TIMEOUT, varStack, "SOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		stream, err = p.bookkeepingClient.CreateRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "StartOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		return
	}
	eorFunc := func(runNumber64 int64) (out string) { // must formally return string even when we return nothing
		callFailedStr := "Bookkeeping EndOfRun call failed"

		bookkeepingDetectorsParam, ok := varStack["bookkeeping_detectors"]
		if !ok {
			log.WithField("partition", envId).
				WithField("runNumber", runNumber64).
				Debug("empty Bookkeeping detectors list provided")
			bookkeepingDetectorsParam = "[\"NULL_DETECTOR\"]"
		}

		detectors, err := p.parseDetectors(bookkeepingDetectorsParam)
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		log.WithField("partition", envId).
			WithField("level", infologger.IL_Ops).
			WithField("runNumber", runNumber64).
			Infof("performing Bookkeeping EOR for detectors: %s", strings.Join(detectors.StringSlice(), " "))

		parameters, ok := varStack["bookkeeping_eor_parameters"]
		if !ok {
			log.WithField("partition", envId).
				Debug("no Bookkeeping EOR parameters set")
			parameters = "{}"
		}

		argMap := make(map[string]string)
		bytes := []byte(parameters)
		err = json.Unmarshal(bytes, &argMap)
		if err != nil {
			err = fmt.Errorf("error processing Bookkeeping EOR parameters: %w", err)

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		// Preparing the per-detector request payload
		in := bkpb.EorRequest{
			RunNumber: int32(runNumber64),
			Detectors: make([]*bkpb.DetectorOperationRequest, len(detectors)),
		}
		for i, det := range detectors {
			perDetectorParameters, ok := varStack[strings.ToLower(det.String())+"_bookkeeping_eor_parameters"]
			if !ok {
				log.WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Debug("empty Bookkeeping detectors list provided")
				perDetectorParameters = "{}"
			}
			detectorArgMap := make(map[string]string)
			bytes := []byte(perDetectorParameters)
			err = json.Unmarshal(bytes, &detectorArgMap)
			if err != nil {
				err = fmt.Errorf("error processing %s Bookkeeping EOR parameter map: %w", det.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					WithField("detector", det.String()).
					WithField("runNumber", runNumber64).
					Error("Bookkeeping error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			// Per-detector parameters override any general bookkeeping_eor_parameters
			err = mergo.Merge(&detectorArgMap, argMap)
			if err != nil {
				err = fmt.Errorf("error processing %s Bookkeeping EOR general parameters override: %w", det.String(), err)

				log.WithError(err).
					WithField("level", infologger.IL_Support).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					WithField("detector", det.String()).
					WithField("runNumber", runNumber64).
					Error("Bookkeeping error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			in.Detectors[i] = &bkpb.DetectorOperationRequest{
				Detector:        det,
				ExtraParameters: detectorArgMap,
			}
		}

		if p.bookkeepingClient == nil {
			err = fmt.Errorf("Bookkeeping plugin not initialized, EndOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		if p.bookkeepingClient.GetConnState() != connectivity.Ready {
			err = fmt.Errorf("Bookkeeping client connection not available, EndOfRun impossible")

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}

		var stream bkpb.Configurator_EndOfRunClient
		timeout := callable.AcquireTimeout(BK_GENERAL_OP_TIMEOUT, varStack, "EOR", envId)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Point of no return
		// The gRPC call below is expected to return immediately, with any actual responses arriving subsequently via
		// the response stream.
		// Regardless of Bookkeeping EOR success or failure, it must run once and only once, therefore if this call returns
		// a nil error, we immediately dequeue the pending EOR.
		stream, err = p.bookkeepingClient.EndOfRun(ctx, &in, grpc.EmptyCallOption{})
		if err != nil {
			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr

			return
		}
		delete(p.pendingEORs, envId) // make sure this EOR never runs again

		log.WithField("level", infologger.IL_Ops).
			WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
			WithField("runNumber", runNumber64).
			WithField("partition", envId).
			WithField("call", "EndOfRun").
			Debug("Bookkeeping EndOfRun returned stream, awaiting responses (Bookkeeping cleanup will not run for this environment)")

		detectorStatusMap := make(map[bkpb.Detector]bkpb.DetectorState)
		for _, v := range detectors {
			detectorStatusMap[v] = bkpb.DetectorState_NULL_STATE
		}

		var bookkeepingEvent *bkpb.RunEvent
		for {
			if ctx.Err() != nil {
				err = fmt.Errorf("Bookkeeping EndOfRun context timed out (%s), any future Bookkeeping events are ignored", timeout.String())
				break
			}
			bookkeepingEvent, err = stream.Recv()
			if errors.Is(err, io.EOF) { // correct stream termination
				log.WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Debug("Bookkeeping EOR event stream EOF, closed")
				break // no more data
			}
			if errors.Is(err, context.DeadlineExceeded) {
				log.WithError(err).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					WithField("timeout", timeout.String()).
					Debug("Bookkeeping EOR timed out")
				err = fmt.Errorf("Bookkeeping EOR timed out after %s: %w", timeout.String(), err)
				break
			}
			if err != nil { // stream termination in case of general error
				log.WithError(err).
					WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Warn("bad Bookkeeping EOR event received, any future Bookkeeping events are ignored")
				break
			}
			if bookkeepingEvent == nil {
				log.WithField("partition", envId).
					WithField("runNumber", runNumber64).
					Warn("nil Bookkeeping EOR event received, skipping to next Bookkeeping event")
				continue
			}

			if bookkeepingEvent.GetState() == bkpb.DetectorState_EOR_FAILURE {
				if err == nil {
					err = fmt.Errorf("%s EOR failure event from Bookkeeping", bookkeepingEvent.GetDetector().String())
				}
				log.WithError(err).
					WithField("event", bookkeepingEvent).
					WithField("detector", bookkeepingEvent.GetDetector().String()).
					WithField("level", infologger.IL_Support).
					WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
					WithField("runNumber", runNumber64).
					WithField("partition", envId).
					WithField("call", "EndOfRun").
					Error("Bookkeeping error")

				call.VarStack["__call_error_reason"] = err.Error()
				call.VarStack["__call_error"] = callFailedStr

				return
			}

			detectorStatusMap[bookkeepingEvent.GetDetector()] = bookkeepingEvent.GetState()

			if bookkeepingEvent.GetState() == bkpb.DetectorState_RUN_OK {
				if bookkeepingEvent.GetDetector() == bkpb.Detector_BK {
					log.WithField("event", bookkeepingEvent).
						WithField("partition", envId).
						WithField("runNumber", runNumber64).
						Debug("Bookkeeping EOR completed successfully")
					delete(p.pendingEORs, envId)
					break
				} else {
					log.WithField("partition", envId).
						WithField("runNumber", runNumber64).
						WithField("detector", bookkeepingEvent.GetDetector().String()).
						Debugf("Bookkeeping EOR for %s: received status %s", bookkeepingEvent.GetDetector().String(), bookkeepingEvent.GetState().String())
				}
			}

			log.WithField("event", bookkeepingEvent).
				WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				WithField("runNumber", runNumber64).
				Info("ALIECS EOR operation : processing Bookkeeping EOR for ")
		}

		bookkeepingFailedDetectors := make([]string, 0)
		bookkeepingopOk := true
		for _, v := range detectors {
			if detectorStatusMap[v] != bkpb.DetectorState_RUN_OK {
				bookkeepingopOk = false
				bookkeepingFailedDetectors = append(bookkeepingFailedDetectors, v.String())
			}
		}
		if bookkeepingopOk {
			delete(p.pendingEORs, envId)
		} else {
			if err == nil {
				err = fmt.Errorf("EOR failed for %s, Bookkeeping EOR will NOT run again for this run", strings.Join(bookkeepingFailedDetectors, ", "))
			}

			log.WithError(err).
				WithField("level", infologger.IL_Support).
				WithField("endpoint", viper.GetString("bookkeepingBaseUri")).
				WithField("runNumber", runNumber64).
				WithField("partition", envId).
				WithField("call", "EndOfRun").
				Error("Bookkeeping error")

			call.VarStack["__call_error_reason"] = err.Error()
			call.VarStack["__call_error"] = callFailedStr
		}
		return
	}
	stack["EndOfRun"] = func() (out string) {
		rn := varStack["run_number"]
		var runNumber64 int64
		var err error
		runNumber64, err = strconv.ParseInt(rn, 10, 32)
		if err != nil {
			log.WithError(err).
				WithField("partition", envId).
				Error("cannot acquire run number for Bookkeeping EOR")
		}
		return eorFunc(runNumber64)
	}
	stack["Cleanup"] = func() (out string) {
		envId, ok := varStack["environment_id"]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Warn("no environment_id found for Bookkeeping cleanup")
			return
		}

		runNumber, ok := p.pendingEORs[envId]
		if !ok {
			log.WithField("partition", envId).
				WithField("level", infologger.IL_Devel).
				Debug("Bookkeeping cleanup: nothing to do")
			return
		}

		log.WithField("runNumber", runNumber).
			WithField("partition", envId).
			WithField("level", infologger.IL_Devel).
			WithField("call", "Cleanup").
			Debug("pending Bookkeeping EOR found, performing cleanup")

		out = eorFunc(runNumber)
		delete(p.pendingEORs, envId)

		return
	}

	return
}

func (p *Plugin) Destroy() error {
	return p.bookkeepingClient.Close()
}
