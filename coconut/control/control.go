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
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/coconut"
	"github.com/AliceO2Group/Control/coconut/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const(
	CALL_TIMEOUT = 55*time.Second
	SPINNER_TICK = 100*time.Millisecond
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type ControlCall func(context.Context, *coconut.RpcClient, *cobra.Command, []string, io.Writer) (error)


func WrapCall(call ControlCall) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("endpoint")
		log.WithPrefix(cmd.Use).
			WithField("endpoint", endpoint).
			Debug("initializing gRPC client")

		s := spinner.New(spinner.CharSets[11], SPINNER_TICK)
		s.Color("yellow")
		s.Suffix = " working..."
		s.Start()

		cxt, cancel := context.WithTimeout(context.Background(), CALL_TIMEOUT)
		rpc := coconut.NewClient(cxt, cancel, endpoint)

		var out strings.Builder

		// redirect stdout to null, the only way to output is
		stdout := os.Stdout
		os.Stdout,_ = os.Open(os.DevNull)
		err := call(cxt, rpc, cmd, args, &out)
		os.Stdout = stdout
		s.Stop()
		if err != nil {
			var fields logrus.Fields
			if logrus.GetLevel() == logrus.DebugLevel {
				fields = logrus.Fields{"error": err}
			}
			log.WithPrefix(cmd.Use).
				WithFields(fields).
				Fatal("command finished with error")
			os.Exit(1)
		}

		fmt.Print(out.String())
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

	_, _ = fmt.Fprintf(o, "instance name:      %s\n", response.GetInstanceName())
	_, _ = fmt.Fprintf(o, "endpoint:           %s\n", green(viper.GetString("endpoint")))
	_, _ = fmt.Fprintf(o, "core version:       %s %s %s\n", response.GetVersion().GetProductName(), versionStr, revisionStr)
	_, _ = fmt.Fprintf(o, "framework id:       %s\n", response.GetFrameworkId())
	_, _ = fmt.Fprintf(o, "environments count: %s\n", green(response.GetEnvironmentsCount()))
	_, _ = fmt.Fprintf(o, "active tasks count: %s\n", green(response.GetTasksCount()))
	_, _ = fmt.Fprintf(o, "global state:       %s\n", colorGlobalState(response.GetState()))

	return nil
}


func Teardown(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	log.Fatal("not implemented yet")
	os.Exit(1)
	return
}


func GetEnvironments(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	var response *pb.GetEnvironmentsReply
	response, err = rpc.GetEnvironments(cxt, &pb.GetEnvironmentsRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	if len(response.GetEnvironments()) == 0 {
		fmt.Fprintln(o, "no environments running")
	} else {
		table := tablewriter.NewWriter(o)
		table.SetHeader([]string{"id", "created", "state"})
		table.SetBorder(false)
		fg := tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor}
		table.SetHeaderColor(fg, fg, fg)

		data := make([][]string, 0, 0)
		for _, envi := range response.GetEnvironments() {
			formatted := formatTimestamp(envi.GetCreatedWhen())
			data = append(data, []string{envi.GetId(), formatted, colorState(envi.GetState())})
		}

		table.AppendBulk(data)
		table.Render()
	}

	return nil
}


func CreateEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	wfPath, err := cmd.Flags().GetString("workflow-template")
	if err != nil {
		return
	}
	if len(wfPath) == 0 {
		err = errors.New("cannot create empty environment")
		return
	}

	var response *pb.NewEnvironmentReply
	response, err = rpc.NewEnvironment(cxt, &pb.NewEnvironmentRequest{WorkflowTemplate: wfPath}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	fmt.Fprintf(o, "new environment created with %s tasks\n", blue(len(tasks)))
	fmt.Fprintf(o, "environment id:     %s\n", grey(env.GetId()))
	fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	fmt.Fprintf(o, "root role:          %s\n", env.GetRootRole())

	return
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

	var response *pb.GetEnvironmentReply
	response, err = rpc.GetEnvironment(cxt, &pb.GetEnvironmentRequest{Id: args[0]}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	rnString := formatRunNumber(env.GetCurrentRunNumber())

	_, _ = fmt.Fprintf(o, "environment id:     %s\n", env.GetId())
	_, _ = fmt.Fprintf(o, "created:            %s\n", formatTimestamp(env.GetCreatedWhen()))
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	_, _ = fmt.Fprintf(o, "run number:         %s\n", rnString)

	if printTasks {
		fmt.Fprintln(o, "")
		drawTableShortTaskInfos(tasks,
			[]string{fmt.Sprintf("task id (%d tasks)", len(tasks)), "class name", "hostname", "status", "state"},
			func(t *pb.ShortTaskInfo) []string {
				return []string{
					t.GetTaskId(),
					t.GetClassName(),
					t.GetDeploymentInfo().GetHostname(),
					t.GetStatus(),
					colorState(t.GetState())}
			}, o)
	}

	if printWorkflow {
		fmt.Fprintf(o, "\nworkflow:\n")
		drawWorkflow(response.GetWorkflow(), o)
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

	var response *pb.ControlEnvironmentReply
	response, err = rpc.ControlEnvironment(cxt, &pb.ControlEnvironmentRequest{Id: args[0], Type: pb.ControlEnvironmentRequest_Optype(pb.ControlEnvironmentRequest_Optype_value[event])}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	rnString := formatRunNumber(response.GetCurrentRunNumber())

	_, _ = fmt.Fprintln(o, "transition complete")
	_, _ = fmt.Fprintf(o, "environment id:     %s\n", response.GetId())
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(response.GetState()))
	_, _ = fmt.Fprintf(o, "run number:         %s\n", rnString)
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
			Type: pb.EnvironmentOperation_ADD_ROLE,
			RoleName: it,
		})
	}
	for _, it := range removeRoles {
		ops = append(ops, &pb.EnvironmentOperation{
			Type: pb.EnvironmentOperation_REMOVE_ROLE,
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
		Id: envId,
		Operations: ops,
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
	fmt.Fprintf(o, "failed operations:  %s\n", failedOpNames )
	return
}


func DestroyEnvironment(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}
	envId := args[0]

	// Check current state first
	var envResponse *pb.GetEnvironmentReply
	envResponse, err = rpc.GetEnvironment(cxt, &pb.GetEnvironmentRequest{Id: envId}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	allowedState := "CONFIGURED"
	if envResponse.GetEnvironment().GetState() != allowedState {
		fmt.Fprint(o, "cannot teardown environment\n")
		fmt.Fprintf(o, "teardown is allowed in state %s, but environment %s is in state %s\n", allowedState, envId, envResponse.GetEnvironment().GetState())
		return
	}

	_, err = rpc.DestroyEnvironment(cxt, &pb.DestroyEnvironmentRequest{Id: envId}, grpc.EmptyCallOption{})
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
			[]string{fmt.Sprintf("task id (%d tasks)", len(tasks)), "class name", "hostname", "locked", "status", "state"},
			func(t *pb.ShortTaskInfo) []string {
				return []string{
					t.GetTaskId(),
					t.GetClassName(),
					t.GetDeploymentInfo().GetHostname(),
					strconv.FormatBool(t.GetLocked()),
					t.GetStatus(),
					colorState(t.GetState())}
			}, o)
	}

	return nil
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
			fmt.Fprintf(o, "(%s)\n", yellow(i))
			fmt.Fprintf(o, "role path:          %s\n", root.GetFullPath())
			fmt.Fprintf(o, "status:             %s\n", root.GetStatus())
			fmt.Fprintf(o, "state:              %s\n", root.GetState())

			fmt.Fprintf(o, "subtree:\n")
			drawWorkflow(root, o)
		}
	}

	return nil
}

func ListWorkflowTemplates(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 0 {
		err = errors.New(fmt.Sprintf("accepts no arg(s), received %d", len(args)))
		return
	}

	var response *pb.GetWorkflowTemplatesReply
	response, err = rpc.GetWorkflowTemplates(cxt, &pb.GetWorkflowTemplatesRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	templates := response.GetWorkflowTemplates()
	if len(templates) == 0 {
		fmt.Fprintln(o, "no templates found")
	} else {
		fmt.Fprintln(o, "available templates in loaded configuration:")
		for _, tmpl := range templates {
			fmt.Fprintln(o, "\t" + tmpl)
		}
	}
	return nil
}

func ListRepos(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 0 {
		err = errors.New(fmt.Sprintf("accepts no args, received %d", len(args)))
		return
	}

	var response *pb.ListReposReply
	response, err = rpc.ListRepos(cxt, &pb.ListReposRequest{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	roots := response.GetRepos()
	if len(roots) == 0 {
		fmt.Fprintln(o, "No repositories found.")
	} else {
		table := tablewriter.NewWriter(o)
		table.SetHeader([]string{"id", "repository", "default"})
		table.SetBorder(false)
		fg := tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlueColor}
		table.SetHeaderColor(fg, fg, fg)

		for i, root := range roots {
			defaultTick := ""
			if root.GetDefault() {
				defaultTick = blue("YES")
			}
			table.Append([]string{strconv.Itoa(i), root.GetName(), defaultTick})
		}
		fmt.Fprintf(o, "Git repositories use the following configuration sources:\n\n")
		table.Render()
	}

	return
}

func AddRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg, received %d", len(args)))
		return
	}

	var response *pb.AddRepoReply
	response, err = rpc.AddRepo(cxt, &pb.AddRepoRequest{Name: args[0]}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	if response.GetErrorString() == "" {
		fmt.Fprintln(o, "Repository succesfully added.")
	} else {
		fmt.Fprintln(o, "Repository couldn't be added:", response.GetErrorString())
	}

	return
}

func RemoveRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg, received %d", len(args)))
		return
	}

	index, _ := strconv.ParseInt(args[0], 10, 32)

	var response *pb.RemoveRepoReply
	response, err = rpc.RemoveRepo(cxt, &pb.RemoveRepoRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	newDefaultRepo := response.GetNewDefaultRepo()
	if response.GetOk() {
		fmt.Fprintln(o, "Repository removed succsefully")
		if newDefaultRepo != "" {
			fmt.Fprintln(o, "New default repo is: " + newDefaultRepo)
		}
	} else {
		fmt.Fprintln(o, "Repository not found")
	}

	return
}

func RefreshRepos(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) > 1 {
		err = errors.New(fmt.Sprintf("accepts 0 or 1 arg(s), received %d", len(args)))
		return
	}

	var response *pb.RefreshReposReply
	if len(args) == 0 {
		response, err = rpc.RefreshRepos(cxt, &pb.RefreshReposRequest{Index: -1}, grpc.EmptyCallOption{})
	} else if len(args) == 1 {
		index, _ := strconv.ParseInt(args[0], 10, 32)

		response, err = rpc.RefreshRepos(cxt, &pb.RefreshReposRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	}

	if err != nil {
		return
	}

	errorString := response.GetErrorString()
	if errorString == "" {
		if len(args) == 0 {
			fmt.Fprintln(o, "Repositories refreshed succesfully")
		} else {
			fmt.Fprintln(o, "Repository refreshed succesfully")
		}
	} else {
		fmt.Fprintln(o, "Repository refresh operation failed:", errorString)
	}

	return
}

func SetDefaultRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg, received %d", len(args)))
		return
	}

	index, _ := strconv.ParseInt(args[0], 10, 32)

	var response *pb.SetDefaultRepoReply
	response, err = rpc.SetDefaultRepo(cxt, &pb.SetDefaultRepoRequest{Index: int32(index)}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	errorString := response.GetErrorString()
	if errorString == "" {
		fmt.Fprintln(o, "Default repository updated succesfully")
	} else {
		fmt.Fprintln(o, "Operation failed:", errorString)
	}

	return
}