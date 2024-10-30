/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

// Package control handles the details of control calls to the O²
// Control core.
package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	commonpb "github.com/AliceO2Group/Control/common/protos"
	"github.com/fatih/color"

	"github.com/xlab/treeprint"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/AliceO2Group/Control/coconut"
	"github.com/AliceO2Group/Control/coconut/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	CALL_TIMEOUT              = 55 * time.Second
	SPINNER_TICK              = 100 * time.Millisecond
	HLCONFIG_COMPONENT_PREFIX = "COG-v1"
	HLCONFIG_PATH_PREFIX      = "consul://o2/runtime/"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type ControlCall func(context.Context, *coconut.RpcClient, *cobra.Command, []string, io.Writer) error

type ConfigurationPayload struct {
	Name       string            `json:"name"`
	Workflow   string            `json:"workflow"`
	Revision   string            `json:"revision"`
	Repository string            `json:"repository"`
	Vars       map[string]string `json:"variables"`
	Detectors  []string          `json:"detectors"`
}

func WrapCall(call ControlCall) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("endpoint")
		log.WithPrefix(cmd.Use).
			WithField("endpoint", endpoint).
			Debug("initializing gRPC client")

		s := spinner.New(spinner.CharSets[11], SPINNER_TICK)
		auto, _ := cmd.Flags().GetBool("auto")
		if !viper.GetBool("nospinner") && !auto {
			_ = s.Color("yellow")
			s.Suffix = " working..."
			s.Start()
		}

		if viper.GetBool("nocolor") {
			color.NoColor = true
		}

		cxt, cancel := context.WithTimeout(context.Background(), CALL_TIMEOUT)
		rpc := coconut.NewClient(cxt, cancel, endpoint)

		var out strings.Builder

		// redirect stdout to null, the only way to output is
		stdout := os.Stdout
		if !auto {
			os.Stdout, _ = os.Open(os.DevNull)
		}
		err := call(cxt, rpc, cmd, args, &out)
		os.Stdout = stdout

		if !viper.GetBool("nospinner") {
			s.Stop()
		}

		fmt.Print(out.String())

		if err != nil {
			log.WithPrefix(cmd.Use).
				WithError(err).
				Fatal("command finished with error")
			os.Exit(1)
		}
	}
}

func GetInfo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	var response *pb.GetFrameworkInfoReply
	response, err = rpc.GetFrameworkInfo(cxt, &pb.GetFrameworkInfoRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	versionStr := response.GetVersion().GetVersionStr()
	// VersionStr will be empty if the core was built with go build directly instead of make.
	// This happens because the Makefile takes care of pushing the version number.
	if len(versionStr) == 0 || versionStr == "0.0.0" {
		versionStr = "dev"
	}
	versionStr = green(versionStr)

	revisionStr := response.GetVersion().GetBuild()
	if len(revisionStr) > 0 {
		revisionStr = fmt.Sprintf("revision %s", green(revisionStr))
	}

	_, _ = fmt.Fprintf(o, "instance name:          %s\n", response.GetInstanceName())
	_, _ = fmt.Fprintf(o, "core version:           %s %s %s\n", response.GetVersion().GetProductName(), versionStr, revisionStr)
	_, _ = fmt.Fprintf(o, "core endpoint:          %s\n", green(viper.GetString("endpoint")))
	_, _ = fmt.Fprintf(o, "configuration endpoint: %s\n", green(response.GetConfigurationEndpoint()))
	_, _ = fmt.Fprintf(o, "framework id:           %s\n", response.GetFrameworkId())
	_, _ = fmt.Fprintf(o, "environments count:     %s\n", green(response.GetEnvironmentsCount()))
	_, _ = fmt.Fprintf(o, "active tasks count:     %s\n", green(response.GetTasksCount()))
	_, _ = fmt.Fprintf(o, "global state:           %s\n", colorGlobalState(response.GetState()))

	allDetectors := response.GetDetectorsInInstance()
	_, _ = fmt.Fprintf(o, "detectors in instance:  %s (%d total)\n", green(strings.Join(allDetectors, " ")), len(allDetectors))
	_, _ = fmt.Fprintf(o, "  active (in use):      %s\n", yellow(strings.Join(response.GetActiveDetectors(), " ")))
	_, _ = fmt.Fprintf(o, "  available:            %s\n", green(strings.Join(response.GetAvailableDetectors(), " ")))

	// Integrated Services API query
	pluginsResponse, err := rpc.GetIntegratedServices(cxt, &pb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	services := pluginsResponse.GetServices()
	enabledCount := 0

	for _, svc := range pluginsResponse.GetServices() {
		if svc.GetEnabled() {
			enabledCount++
		}
	}
	intservMessage := "none"
	if len(services) > 0 {
		intservMessage = fmt.Sprintf("%s enabled (out of %d total)", green(enabledCount), len(services))
	}

	_, _ = fmt.Fprintf(o, "integrated services:    %s\n", intservMessage)

	sortedSvcIds := make([]string, len(services))
	i := 0
	for svcId := range services {
		sortedSvcIds[i] = svcId
		i++
	}
	sort.Strings(sortedSvcIds)

	for _, svcId := range sortedSvcIds {
		svc := services[svcId]
		enabledString := dark("disabled")
		if svc.Enabled {
			enabledString = green("enabled")
		}

		_, _ = fmt.Fprintf(o, "  %-22s%s\n", svcId+":", enabledString)
		if !svc.Enabled {
			continue
		}

		connectionState := svc.GetConnectionState()
		svcEndpoint := svc.GetEndpoint()
		switch connectionState {
		case "TRANSIENT_FAILURE":
			connectionState = red(connectionState)
			svcEndpoint = red(svcEndpoint)
		case "CONNECTING":
			connectionState = yellow(connectionState)
			svcEndpoint = yellow(svcEndpoint)
		case "READY":
			connectionState = green(connectionState)
			svcEndpoint = green(svcEndpoint)
		}

		_, _ = fmt.Fprintf(o, "    %-20s%s\n", "service name:", svc.GetName())
		_, _ = fmt.Fprintf(o, "    %-20s%s\n", "endpoint:", svcEndpoint)
		_, _ = fmt.Fprintf(o, "    %-20s%s\n", "connection state:", connectionState)
		if svcData := svc.GetData(); svcData != "{}" {
			_, _ = fmt.Fprintf(o, "    %-20s%s\n", "data:", svcData)
		}
	}

	return nil
}

func Teardown(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	log.Fatal("not implemented yet")
	os.Exit(1)
	return
}

func GetEnvironments(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	var response *pb.GetEnvironmentsReply
	showAll, _ := cmd.Flags().GetBool("show-all")
	response, err = rpc.GetEnvironments(cxt, &pb.GetEnvironmentsRequest{ShowAll: showAll}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	if len(response.GetEnvironments()) == 0 {
		fmt.Fprintln(o, "no environments running")
	} else {
		table := tablewriter.NewWriter(o)
		table.SetHeader([]string{"id", "workflow template", "created", "state"})
		table.SetBorder(false)
		fg := tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor}
		table.SetHeaderColor(fg, fg, fg, fg)

		data := make([][]string, 0, 0)
		for _, envi := range response.GetEnvironments() {
			formatted := formatTimestamp(envi.GetCreatedWhen())
			data = append(data, []string{envi.GetId(), envi.GetRootRole(), formatted, colorState(envi.GetState())})
		}

		table.AppendBulk(data)
		table.Render()
	}

	return nil
}

func CreateEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	configPayloadPath, err := cmd.Flags().GetString("configuration")
	if err != nil {
		return fmt.Errorf("cannot get configuration payload path value: %w", err)
	}
	userWfPath, err := cmd.Flags().GetString("workflow-template")
	if err != nil {
		return fmt.Errorf("cannot get workflow template path value: %w", err)
	}
	if cmd.Flags().Changed("configuration") && len(configPayloadPath) == 0 && cmd.Flags().Changed("workflow-template") && len(userWfPath) == 0 {
		return fmt.Errorf("no configuration payload path or workflow template path provided: %w", err)
	}

	payloadData := new(ConfigurationPayload)
	var wfPath string
	if cmd.Flags().Changed("configuration") && len(configPayloadPath) > 0 {
		if strings.HasSuffix(strings.ToLower(configPayloadPath), ".json") {
			configPayloadDoc, err := os.ReadFile(configPayloadPath)
			if err != nil {
				return fmt.Errorf("cannot read local configuration file %v: %w", configPayloadPath, err)
			}
			err = json.Unmarshal(configPayloadDoc, &payloadData)
			if err != nil {
				return fmt.Errorf("cannot unmarshal local configuration file %v: %w", configPayloadPath, err)
			}
		} else {
			if strings.HasPrefix(configPayloadPath, HLCONFIG_PATH_PREFIX+HLCONFIG_COMPONENT_PREFIX) {
				configPayloadPath = strings.TrimPrefix(configPayloadPath, HLCONFIG_PATH_PREFIX+HLCONFIG_COMPONENT_PREFIX)
			}
			configPayloadDoc, err := apricot.Instance().GetRuntimeEntry(HLCONFIG_COMPONENT_PREFIX, configPayloadPath)
			if err != nil {
				return fmt.Errorf("cannot retrieve file %v from "+HLCONFIG_PATH_PREFIX+HLCONFIG_COMPONENT_PREFIX+": %w", configPayloadPath, err)
			}
			err = json.Unmarshal([]byte(configPayloadDoc), &payloadData)
			if err != nil {
				return fmt.Errorf("cannot unmarshal remote configuration payload %v: %w", configPayloadPath, err)
			}
		}

		if cmd.Flags().Changed("workflow-template") && len(userWfPath) == 0 {
			if len(payloadData.Workflow) > 0 {
				wfPath = payloadData.Workflow
			} else {
				return errors.New("empty workflow template in configuration payload, and empty workflow template path provided")
			}
		} else if cmd.Flags().Changed("workflow-template") && len(userWfPath) > 0 {
			wfPath = userWfPath
		} else {
			if len(payloadData.Workflow) > 0 {
				wfPath = payloadData.Workflow
			} else {
				return errors.New("no workflow template provided in configuration payload")
			}
		}
	} else {
		if cmd.Flags().Changed("workflow-template") && len(userWfPath) > 0 {
			wfPath = userWfPath
		} else {
			return errors.New("empty workflow template path provided")
		}
	}

	var extraVars string
	extraVars, err = cmd.Flags().GetString("extra-vars")
	if err != nil {
		return
	}

	extraVars = strings.TrimSpace(extraVars)
	if cmd.Flags().Changed("extra-vars") && len(extraVars) == 0 {
		err = errors.New("empty list of extra-vars supplied")
		return
	}

	userExtraVarsMap, err := utils.ParseExtraVars(extraVars)
	if err != nil {
		return
	}

	extraVarsMap := make(map[string]string, 0)
	for k, v := range userExtraVarsMap {
		extraVarsMap[k] = v
	}
	if cmd.Flags().Changed("configuration") && len(configPayloadPath) > 0 {
		for k, v := range payloadData.Vars {
			if _, exists := extraVarsMap[k]; !exists {
				extraVarsMap[k] = v
			}
		}
	}

	public, _ := cmd.Flags().GetBool("public")

	auto, _ := cmd.Flags().GetBool("auto")
	if auto {
		// subscribe to core to receive events
		id := xid.New().String()
		stream, err := rpc.Subscribe(context.TODO(), &pb.SubscribeRequest{Id: id}, grpc.EmptyCallOption{})
		if err != nil {
			log.WithPrefix("Subscribe").
				WithError(err).
				Fatal("command finished with error")
		}
		_, err = rpc.NewAutoEnvironment(cxt, &pb.NewAutoEnvironmentRequest{
			WorkflowTemplate: wfPath,
			Vars:             extraVarsMap,
			Id:               id,
			RequestUser: &commonpb.User{
				Name: getUserAndHost(),
			}}, grpc.EmptyCallOption{})
		if err != nil {
			return err
		}
		for {
			rcv, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.WithPrefix("Subscribe").
					WithError(err).
					Fatal("command finished with error")
			}
			if evt := rcv.GetEnvironmentEvent(); evt != nil {
				if evt.Error != "" {
					if viper.GetBool("verbose") {
						log.WithPrefix("Event").
							WithError(fmt.Errorf(evt.Error)).
							Error(evt.EnvironmentId)
						return nil
					}
					fmt.Printf("\nEnvironment with id %s failed with error: %s\n", evt.EnvironmentId, evt.Error)
				} else {
					tmpl, err := template.New("envEvents").Parse("Enviroment {{.EnvironmentId}} {{if .Message}}{{.Message}} {{end}}{{if .State}}changed state to {{.State}}{{end}}{{if .CurrentRunNumber}} with run number {{.CurrentRunNumber}}{{end}}\n")
					if err != nil {
						return err
					}
					err = tmpl.Execute(os.Stdout, evt)
					if err != nil {
						return err
					}
				}
			}
			if viper.GetBool("verbose") {
				log.WithPrefix("Event").
					Info(rcv)
			}
			if evt := rcv.GetTaskEvent(); evt != nil {
				tmpl, err := template.New("taskEvents").Parse("Task {{.Taskid}} of class {{.ClassName}} changed{{if .State}} state to {{.State}}{{end}}{{if .Status}} status to {{.Status}}{{end}} on machine {{.Hostname}}\n")
				if err != nil {
					return err
				}
				err = tmpl.Execute(os.Stdout, evt)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	// TODO: add support for setting visibility here OCTRL-178
	// TODO: add support for acquiring bot config here OCTRL-177

	asynchronous, _ := cmd.Flags().GetBool("asynchronous")

	var response *pb.NewEnvironmentReply
	if asynchronous {
		response, err = rpc.NewEnvironmentAsync(cxt, &pb.NewEnvironmentRequest{
			WorkflowTemplate: wfPath,
			Vars:             extraVarsMap,
			Public:           public,
			RequestUser: &commonpb.User{
				Name: getUserAndHost(),
			},
		}, grpc.EmptyCallOption{})
	} else {
		response, err = rpc.NewEnvironment(cxt, &pb.NewEnvironmentRequest{
			WorkflowTemplate: wfPath,
			Vars:             extraVarsMap,
			Public:           public,
			RequestUser: &commonpb.User{
				Name: getUserAndHost(),
			},
		}, grpc.EmptyCallOption{})
	}
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	_, _ = fmt.Fprintf(o, "new environment created with %s tasks\n", blue(len(tasks)))
	_, _ = fmt.Fprintf(o, "environment id:     %s\n", grey(env.GetId()))
	_, _ = fmt.Fprintf(o, "workflow template:  %s\n", env.GetRootRole())
	_, _ = fmt.Fprintf(o, "description:        %s\n", env.GetDescription())
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	_, _ = fmt.Fprintf(o, "public:             %v\n", response.Public)
	_, _ = fmt.Fprintf(o, "tasks active:       %v\n", env.GetNumberOfActiveTasks())
	_, _ = fmt.Fprintf(o, "tasks inactive:     %v\n", env.GetNumberOfInactiveTasks())
	_, _ = fmt.Fprintf(o, "tasks total:        %v\n", env.GetNumberOfTasks())

	var (
		defaultsStr = stringMapToString(env.Defaults, "\t")
		varsStr     = stringMapToString(env.Vars, "\t")
		userVarsStr = stringMapToString(env.UserVars, "\t")
	)
	if len(defaultsStr) != 0 {
		_, _ = fmt.Fprintf(o, "global defaults:\n%s\n", defaultsStr)
	}
	if len(varsStr) != 0 {
		_, _ = fmt.Fprintf(o, "global variables:\n%s\n", varsStr)
	}
	if len(userVarsStr) != 0 {
		_, _ = fmt.Fprintf(o, "user-provided variables:\n%s\n", userVarsStr)
	}

	return
}

func stringMapToString(stringMap map[string]string, indent string) string {
	if len(stringMap) == 0 {
		return ""
	}
	accumulator := make(sort.StringSlice, len(stringMap))
	i := 0
	for k, v := range stringMap {
		value := v
		if len(v) == 0 {
			value = "<empty>"
		}
		accumulator[i] = indent + k + ": " + value
		i++
	}
	accumulator.Sort()
	return strings.Join(accumulator, "\n")
}

func ShowEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}

	printTasks, err := cmd.Flags().GetBool("tasks")
	if err != nil {
		return
	}
	printWorkflow, err := cmd.Flags().GetBool("workflow")
	if err != nil {
		return
	}
	printIntegratedServicesData, err := cmd.Flags().GetBool("services")
	if err != nil {
		return
	}

	var response *pb.GetEnvironmentReply
	response, err = rpc.GetEnvironment(cxt, &pb.GetEnvironmentRequest{Id: args[0], ShowWorkflowTree: true}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	rnString := formatRunNumber(env.GetCurrentRunNumber())

	var (
		defaultsStr = stringMapToString(env.Defaults, "\t")
		varsStr     = stringMapToString(env.Vars, "\t")
		userVarsStr = stringMapToString(env.UserVars, "\t")
	)

	_, _ = fmt.Fprintf(o, "environment id:     %s\n", env.GetId())
	_, _ = fmt.Fprintf(o, "workflow template:  %s\n", env.GetRootRole())
	_, _ = fmt.Fprintf(o, "description:        %s\n", env.GetDescription())
	_, _ = fmt.Fprintf(o, "created:            %s\n", formatTimestamp(env.GetCreatedWhen()))
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	if currentTransition := env.GetCurrentTransition(); len(currentTransition) != 0 {
		_, _ = fmt.Fprintf(o, "transition:              %s\n", currentTransition)
	}
	_, _ = fmt.Fprintf(o, "public:             %t\n", response.Public)
	_, _ = fmt.Fprintf(o, "run number:         %s\n", rnString)
	_, _ = fmt.Fprintf(o, "number of FLPs:     %s\n", formatNumber(env.GetNumberOfFlps()))
	_, _ = fmt.Fprintf(o, "detectors:          %s\n", strings.Join(response.GetEnvironment().GetIncludedDetectors(), " "))
	if len(defaultsStr) != 0 {
		_, _ = fmt.Fprintf(o, "global defaults:\n%s\n", defaultsStr)
	}
	if len(varsStr) != 0 {
		_, _ = fmt.Fprintf(o, "global variables:\n%s\n", varsStr)
	}
	if len(userVarsStr) != 0 {
		_, _ = fmt.Fprintf(o, "user-provided variables:\n%s\n", userVarsStr)
	}

	if printTasks {
		_, _ = fmt.Fprintln(o, "")
		drawTableShortTaskInfos(tasks,
			[]string{fmt.Sprintf("task id (%d)", len(tasks)), "class name", "hostname", "crit", "status", "state"},
			func(t *pb.ShortTaskInfo) []string {
				return []string{
					t.GetTaskId(),
					t.GetClassName(),
					t.GetDeploymentInfo().GetHostname(),
					func() string {
						if t.GetCritical() {
							return "YES"
						}
						return "NO"
					}(),
					t.GetStatus(),
					colorState(t.GetState())}
			}, o)
	}

	if printWorkflow {
		_, _ = fmt.Fprintf(o, "\nworkflow:\n")
		drawWorkflow(response.GetWorkflow(), o)
	}

	if printIntegratedServicesData {
		_, _ = fmt.Fprintf(o, "\nintegrated services:\n")
		drawIntegratedServicesData(response.GetEnvironment().GetIntegratedServicesData(), o)
	}
	return
}

func ControlEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}

	event, err := cmd.Flags().GetString("event")
	if err != nil {
		return
	}
	event = strings.ToUpper(event)
	switch event {
	case "START":
		fallthrough
	case "STOP":
		event = event + "_ACTIVITY"
	}

	var response *pb.ControlEnvironmentReply
	response, err = rpc.ControlEnvironment(cxt, &pb.ControlEnvironmentRequest{
		Id:   args[0],
		Type: pb.ControlEnvironmentRequest_Optype(pb.ControlEnvironmentRequest_Optype_value[event]),
		RequestUser: &commonpb.User{
			Name: getUserAndHost(),
		},
	}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	rnString := formatRunNumber(response.GetCurrentRunNumber())

	sotTimestamp := time.Unix(0, response.GetStartOfTransition()*int64(time.Millisecond))
	sotFormatted := sotTimestamp.Local().Format("2006-01-02 15:04:05.000000 MST")
	eotTimestamp := time.Unix(0, response.GetEndOfTransition()*int64(time.Millisecond))
	eotFormatted := eotTimestamp.Local().Format("2006-01-02 15:04:05.000000 MST")
	tdTimestamp := time.Duration(response.GetTransitionDuration() * int64(time.Millisecond))

	_, _ = fmt.Fprintln(o, "transition complete")
	_, _ = fmt.Fprintf(o, "environment id:      %s\n", response.GetId())
	_, _ = fmt.Fprintf(o, "state:               %s\n", colorState(response.GetState()))
	_, _ = fmt.Fprintf(o, "run number:          %s\n", rnString)
	_, _ = fmt.Fprintf(o, "start of transition: %s\n", grey(sotFormatted))
	_, _ = fmt.Fprintf(o, "end of transition:   %s\n", grey(eotFormatted))
	_, _ = fmt.Fprintf(o, "transition duration: %ss\n", grey(strconv.FormatFloat(tdTimestamp.Seconds(), 'f', 6, 64)))
	return
}

func ModifyEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}
	envId := args[0]

	addRoles, err := cmd.Flags().GetStringArray("addroles")
	if err != nil {
		fmt.Fprintln(o, "error: addroles")
		return
	}

	removeRoles, err := cmd.Flags().GetStringArray("removeroles")
	if err != nil {
		fmt.Fprintln(o, "error: removeroles")
		return
	}

	reconfigure, err := cmd.Flags().GetBool("reconfigure")
	if err != nil {
		fmt.Fprintln(o, "error: reconfigure")
		return
	}

	ops := make([]*pb.EnvironmentOperation, 0)
	for _, it := range addRoles {
		ops = append(ops, &pb.EnvironmentOperation{
			Type:     pb.EnvironmentOperation_ADD_ROLE,
			RoleName: it,
		})
	}
	for _, it := range removeRoles {
		ops = append(ops, &pb.EnvironmentOperation{
			Type:     pb.EnvironmentOperation_REMOVE_ROLE,
			RoleName: it,
		})
	}

	if len(ops) == 0 {
		fmt.Fprintln(o, "no changes requested")
		return
	}

	// Check current state first
	var envResponse *pb.GetEnvironmentReply
	envResponse, err = rpc.GetEnvironment(cxt, &pb.GetEnvironmentRequest{Id: envId}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	allowedState := "CONFIGURED"
	if envResponse.GetEnvironment().GetState() != allowedState {
		fmt.Fprint(o, "cannot modify environment\n")
		fmt.Fprintf(o, "workflow changes are allowed in state %s, but environment %s is in state %s\n", allowedState, envId, envResponse.GetEnvironment().GetState())
		return
	}

	// Do the request
	var response *pb.ModifyEnvironmentReply
	response, err = rpc.ModifyEnvironment(cxt, &pb.ModifyEnvironmentRequest{
		Id:             envId,
		Operations:     ops,
		ReconfigureAll: reconfigure,
	}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	fmt.Fprintln(o, "environment modified")
	fmt.Fprintf(o, "environment id:     %s\n", response.GetId())
	fmt.Fprintf(o, "state:              %s\n", response.GetState())

	failedOps := response.GetFailedOperations()
	failedOpNames := func() (f []string) {
		for _, v := range failedOps {
			f = append(f, fmt.Sprintf("%s: %s", pb.EnvironmentOperation_Optype_name[int32(v.GetType())], v.GetRoleName()))
		}
		return
	}()
	fmt.Fprintf(o, "failed operations:  %s\n", failedOpNames)
	return
}

func DestroyEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}
	envId := args[0]

	keepTasks, err := cmd.Flags().GetBool("keep-tasks")
	if err != nil {
		keepTasks = false
	}

	allowInRunningState, err := cmd.Flags().GetBool("allow-in-running-state")
	if err != nil {
		keepTasks = false
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		keepTasks = false
	}

	_, err = rpc.DestroyEnvironment(cxt, &pb.DestroyEnvironmentRequest{
		Id:                  envId,
		KeepTasks:           keepTasks,
		AllowInRunningState: allowInRunningState,
		Force:               force,
		RequestUser: &commonpb.User{
			Name: getUserAndHost(),
		},
	}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	fmt.Fprintf(o, "teardown complete for environment %s\n", envId)

	return
}

func GetTasks(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	var response *pb.GetTasksReply
	response, err = rpc.GetTasks(cxt, &pb.GetTasksRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	tasks := response.GetTasks()

	if len(tasks) == 0 {
		fmt.Fprintln(o, "no tasks running")
	} else {
		drawTableShortTaskInfos(tasks,
			[]string{fmt.Sprintf("task id (%d)", len(tasks)), "class name", "hostname", "locked", "claimable", "crit", "status", "state", "PID"},
			func(t *pb.ShortTaskInfo) []string {
				return []string{
					t.GetTaskId(),
					t.GetClassName(),
					t.GetDeploymentInfo().GetHostname(),
					strconv.FormatBool(t.GetLocked()),
					strconv.FormatBool(t.GetClaimable()),
					func() string {
						if t.GetCritical() {
							return "YES"
						}
						return "NO"
					}(),
					t.GetStatus(),
					colorState(t.GetState()),
					t.GetPid(),
				}
			}, o)
	}

	return nil
}

func CleanTasks(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) > 0 {
		for _, id := range args {
			if len(id) == 0 {
				err = errors.New(fmt.Sprintf("%s is not a valid task ID", id))
				return
			}
		}
	}
	var response *pb.CleanupTasksReply
	response, err = rpc.CleanupTasks(cxt, &pb.CleanupTasksRequest{TaskIds: args}, grpc.EmptyCallOption{})
	if err != nil && response == nil {
		return
	}

	if len(response.KilledTasks) == 0 {
		fmt.Fprintln(o, "0 tasks killed")
	} else {
		drawTableShortTaskInfos(response.KilledTasks,
			[]string{fmt.Sprintf("task id (%d killed)", len(response.KilledTasks)), "class name", "hostname"},
			func(t *pb.ShortTaskInfo) []string {
				return []string{
					t.GetTaskId(),
					t.GetClassName(),
					t.GetDeploymentInfo().GetHostname()}
			}, o)
	}

	_, _ = fmt.Fprintf(o, "%d tasks running\n", len(response.RunningTasks))

	return
}

func QueryRoles(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 2 {
		err = errors.New(fmt.Sprintf("accepts 2 arg(s), received %d", len(args)))
		return
	}
	envId := args[0]
	queryPath := args[1]

	var response *pb.GetRolesReply
	response, err = rpc.GetRoles(cxt, &pb.GetRolesRequest{EnvId: envId, PathSpec: queryPath}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	roots := response.GetRoles()

	if len(roots) == 0 {
		fmt.Fprintln(o, "no roles found")
	} else {
		for i, root := range roots {
			var (
				defaultsStr     = stringMapToString(root.Defaults, "\t")
				varsStr         = stringMapToString(root.Vars, "\t")
				userVarsStr     = stringMapToString(root.UserVars, "\t")
				consolidatedStr = stringMapToString(root.ConsolidatedStack, "\t")
			)

			_, _ = fmt.Fprintf(o, "(%s)\n", yellow(i))
			_, _ = fmt.Fprintf(o, "role path:          %s\n", root.GetFullPath())
			_, _ = fmt.Fprintf(o, "status:             %s\n", root.GetStatus())
			_, _ = fmt.Fprintf(o, "state:              %s\n", root.GetState())
			if len(defaultsStr) != 0 {
				_, _ = fmt.Fprintf(o, "role defaults:\n%s\n", defaultsStr)
			}
			if len(varsStr) != 0 {
				_, _ = fmt.Fprintf(o, "role variables:\n%s\n", varsStr)
			}
			if len(userVarsStr) != 0 {
				_, _ = fmt.Fprintf(o, "user-provided variables:\n%s\n", userVarsStr)
			}
			if len(consolidatedStr) != 0 {
				_, _ = fmt.Fprintf(o, "consolidated stack:\n%s\n", consolidatedStr)
			}
			_, _ = fmt.Fprintf(o, "subtree:\n")
			drawWorkflow(root, o)
		}
	}

	return nil
}

// ListWorkflowTemplates lists the available workflow templates and the git repo on which they reside.
func ListWorkflowTemplates(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	repoPattern := ""
	revisionPattern := "" // Let the API take care of defaults
	allBranches := false
	allTags := false
	allWorkflows := false
	showDescription := false

	if len(args) == 0 {
		repoPattern, err = cmd.Flags().GetString("repository")
		if err != nil {
			return
		}

		revisionPattern, err = cmd.Flags().GetString("revision")
		if err != nil {
			return
		}

		allBranches, err = cmd.Flags().GetBool("all-branches")
		if err != nil {
			return
		}

		allTags, err = cmd.Flags().GetBool("all-tags")
		if err != nil {
			return
		}

		allWorkflows, err = cmd.Flags().GetBool("all-workflows")
		if err != nil {
			return
		}

		showDescription, err = cmd.Flags().GetBool("show-description")

		if allBranches || allTags {
			if revisionPattern != "" {
				fmt.Fprintln(o, "Ignoring `--all-{branches,tags}` flags, as a valid revision has been specified")
				allBranches = false
				allTags = false
			}
		}

	} else if len(args) == 1 { // If we have an argument, give priority over the flags
		simpleRepoRegex := regexp.MustCompile("\\A[^@]+\\z")
		repoRevisionRegex := regexp.MustCompile("\\A[^@]+@[^@]+\\z")

		if simpleRepoRegex.MatchString(args[0]) {
			repoPattern = args[0]
		} else if repoRevisionRegex.MatchString(args[0]) {
			slicedArgument := strings.Split(args[0], "@")
			repoPattern = slicedArgument[0]
			revisionPattern = slicedArgument[1]
		} else {
			err = errors.New("arguments should be in the form of [repo-pattern](@[revision-pattern])")
			return
		}

		if checkForFlag, _ := cmd.Flags().GetString("repository"); checkForFlag != "*" { // "*" comes from the flag's default value
			fmt.Fprintln(o, "Ignoring `--repo` flag, as a valid argument has been passed ")
		}

		if checkForFlag, _ := cmd.Flags().GetString("revision"); checkForFlag != "master" {
			fmt.Fprintln(o, "Ignoring `--revision` flag, as a valid argument has been passed")
		}

		if checkForFlag, _ := cmd.Flags().GetBool("all-branches"); checkForFlag != false {
			fmt.Fprintln(o, "Ignoring `--all-branches` flag, as a valid argument has been passed")
		}

		if checkForFlag, _ := cmd.Flags().GetBool("all-tags"); checkForFlag != false {
			fmt.Fprintln(o, "Ignoring `--all-tags` flag, as a valid argument has been passed")
		}

	} else {
		err = errors.New(fmt.Sprintf("expecting one argument or a combination of the --repo and --revision flags, %d args received", len(args)))

		return
	}

	var response *pb.GetWorkflowTemplatesReply
	response, err = rpc.GetWorkflowTemplates(cxt, &pb.GetWorkflowTemplatesRequest{RepoPattern: repoPattern, RevisionPattern: revisionPattern,
		AllBranches: allBranches, AllTags: allTags, AllWorkflows: allWorkflows}, grpc.EmptyCallOption{})
	if err != nil {
		return err
	}

	templates := response.GetWorkflowTemplates()
	if len(templates) == 0 {
		fmt.Fprintln(o, "No templates found.")
	} else {
		var prevRepo string
		var prevRevision string
		aTree := treeprint.New()
		aTree.SetValue("Available templates in loaded configuration:")

		revBranch := treeprint.New()

		for _, tmpl := range templates {
			if prevRepo != tmpl.GetRepo() { // Create the root node of the tree w/ the repo name
				fmt.Fprint(o, aTree.String())
				aTree = treeprint.New()
				aTree.SetValue(blue(tmpl.GetRepo()))
				prevRepo = tmpl.GetRepo()
				prevRevision = "" // Reinitialize the previous revision
			}

			if prevRevision != tmpl.GetRevision() { // Create the first leaf of the root node w/ the revision name
				revBranch = aTree.AddBranch(tmpl.GetRevision)
				revBranch.SetValue(yellow("[revision] " + tmpl.GetRevision())) // Otherwise the pointer value was set as the branch's value
				prevRevision = tmpl.GetRevision()
			}
			if showDescription {
				tmplBranch := revBranch.AddBranch(tmpl.GetTemplate())
				tmplBranch.AddNode(grey(tmpl.GetDescription()))
			} else {
				revBranch.AddNode(tmpl.GetTemplate())
			}
		}

		fmt.Fprint(o, aTree.String())
	}

	return nil
}

// ListRepos lists all available git repositories that are used for configuration.
func ListRepos(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 0 {
		err = errors.New(fmt.Sprintf("accepts no args, received %d", len(args)))
		return err
	}

	var response *pb.ListReposReply
	response, err = rpc.ListRepos(cxt, &pb.ListReposRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return err
	}

	roots := response.GetRepos()
	if len(roots) == 0 {
		fmt.Fprintln(o, "No repositories found.")
	} else {
		table := tablewriter.NewWriter(o)
		table.SetHeader([]string{"id", "repository", "default", "default revision"})
		table.SetBorder(false)
		fg := tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlueColor}
		table.SetHeaderColor(fg, fg, fg, fg)

		globalDefaultRevision := response.GetGlobalDefaultRevision()

		for i, root := range roots {
			defaultTick := ""

			if root.GetDefault() {
				defaultTick = blue("YES")
			}
			var defaultRevision string
			if root.GetDefaultRevision() == globalDefaultRevision {
				defaultRevision = red(globalDefaultRevision)
			} else {
				defaultRevision = root.GetDefaultRevision()
			}
			table.Append([]string{strconv.Itoa(i), root.GetName(), defaultTick, defaultRevision})
		}
		fmt.Fprintf(o, "Git repositories used as configuration sources:\n\n")
		table.Render()
		fmt.Fprintf(o, "\nGlobal default revision: %s\n", red(globalDefaultRevision))
	}

	return nil
}

// AddRepo add a new repository to the available git repositories used for configuration and checks it out.
func AddRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) error {

	if len(args) != 1 {
		return fmt.Errorf("accepts 1 arg, received %d", len(args))
	}

	name := args[0]
	defaultRevision, err := cmd.Flags().GetString("default-revision")
	if err != nil {
		return err
	}

	response, err := rpc.AddRepo(cxt, &pb.AddRepoRequest{Name: name, DefaultRevision: defaultRevision}, grpc.EmptyCallOption{})
	if err != nil {
		fmt.Fprintln(o, "Cannot add repository.")
		return err
	}

	fmt.Fprintln(o, "Repository succesfully added.")
	fmt.Fprintln(o, response.GetInfo())

	return nil
}

// RemoveRepo removes a git repository based on the indexes reported by ListRepos.
// If the default repository is removed, the repository with the lowest index is
// set as the new default. If all repositories are removed, the backend (consul or file)
// record of the default repository is updated to the relevant viper entry.
func RemoveRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg, received %d", len(args)))
		return err
	}

	index, _ := strconv.ParseInt(args[0], 10, 32)

	var response *pb.RemoveRepoReply
	response, err = rpc.RemoveRepo(cxt, &pb.RemoveRepoRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	if err != nil {
		return err
	}

	newDefaultRepo := response.GetNewDefaultRepo()
	fmt.Fprintln(o, "Repository removed successfully.")
	if newDefaultRepo != "" {
		fmt.Fprintln(o, "New default repo is: "+newDefaultRepo)
	}

	return nil
}

// RefreshRepos runs the equivalent of git pull origin/master for all available repositories.
func RefreshRepos(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) > 1 {
		err = errors.New(fmt.Sprintf("accepts 0 or 1 arg(s), received %d", len(args)))
		return err
	}

	if len(args) == 0 {
		_, err = rpc.RefreshRepos(cxt, &pb.RefreshReposRequest{Index: -1}, grpc.EmptyCallOption{})
	} else if len(args) == 1 {
		index, _ := strconv.ParseInt(args[0], 10, 32)

		_, err = rpc.RefreshRepos(cxt, &pb.RefreshReposRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	}

	if err != nil {
		fmt.Fprintln(o, "Repository refresh operation failed.")
		return err
	}

	if len(args) == 0 {
		fmt.Fprintln(o, "Repositories refreshed succesfully")
	} else {
		fmt.Fprintln(o, "Repository refreshed succesfully")
	}

	return nil
}

// SetDefaultRepo selects the default repository based on the indexes reported by ListRepos.
// It also updates the backend (consul or file) which holds a record for the default repository
// which is persistent across core executions.
func SetDefaultRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) error {
	if len(args) != 1 {
		err := errors.New(fmt.Sprintf("accepts 1 arg, received %d", len(args)))
		return err
	}

	index, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		fmt.Fprintln(o, "Wrong argument; should be repository's index")
		return err
	}

	_, err = rpc.SetDefaultRepo(cxt, &pb.SetDefaultRepoRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	if err != nil {
		fmt.Fprintln(o, "Operation failed.")
		return err
	}

	fmt.Fprintln(o, "Default repository update succesfully")

	return nil
}

// SetDefaultRevision selects the default repository revision.
// This can be done on the global or on the repository level.
func SetDefaultRevision(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) error {
	if len(args) == 1 { // Set global default
		_, err := rpc.SetGlobalDefaultRevision(cxt, &pb.SetGlobalDefaultRevisionRequest{Revision: args[0]}, grpc.EmptyCallOption{})
		if err != nil {
			fmt.Fprintln(o, "Operation failed.")
			return err
		}
		fmt.Fprintln(o, "The global default revision has been succesfuly updated to \""+args[0]+"\".")
	} else if len(args) == 2 { // Set per-repo default
		index, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			fmt.Fprintln(o, "Wrong argument; should be repository's index")
			return err
		}

		var response *pb.SetRepoDefaultRevisionReply
		response, err = rpc.SetRepoDefaultRevision(cxt, &pb.SetRepoDefaultRevisionRequest{Index: int32(index), Revision: args[1]}, grpc.EmptyCallOption{})
		if err != nil {
			fmt.Fprintln(o, "Operation failed.")
			return err
		} else if response.GetInfo() != "" {
			fmt.Fprintln(o, "Operation failed.\n")
			fmt.Fprintln(o, "Available revisions for this repo: \n"+response.GetInfo())
			return errors.New("Could not update the default revision.")
		}
		fmt.Fprintln(o, "The default revision for this repo has been successfuly updated to \""+args[1]+"\".")
	} else {
		err := errors.New(fmt.Sprintf("expects 1 or 2 args, received %d", len(args)))
		return err
	}

	return nil
}
