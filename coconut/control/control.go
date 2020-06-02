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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xlab/treeprint"
	"gopkg.in/yaml.v3"

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
		_ = s.Color("yellow")
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
		table.SetHeader([]string{"id", "root role", "created", "state"})
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


func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

func isJson(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
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

	extraVarsMap := make(map[string]string)

	if isJson(extraVars) {
		extraVarsMapI := make(map[string]interface{})
		err = yaml.Unmarshal([]byte(extraVars), &extraVarsMapI)
		if err != nil {
			err = fmt.Errorf("cannot parse extra-vars as JSON: %w", err)
			return
		}
		for k, v := range extraVarsMapI {
			if strVal, ok := v.(string); ok {
				extraVarsMap[k] = strVal
				continue
			}
			marshaledValue, marshalErr := json.Marshal(v)
			if marshalErr != nil {
				continue
			}
			extraVarsMap[k] = string(marshaledValue)
		}
	} else {
		extraVarsSlice := make([]string, 0)
		extraVarsSlice, err = readAsCSV(extraVars)
		if err != nil {
			err = fmt.Errorf("cannot parse extra-vars as CSV: %w", err)
			return
		}

		for _, entry := range extraVarsSlice {
			if len(entry) < 3 { // can't be shorter than a=b
				err = fmt.Errorf("invalid variable assignment %s", entry)
				return
			}
			if strings.Count(entry, "=") != 1 {
				err = fmt.Errorf("invalid variable assignment %s", entry)
				return
			}

			sanitized := strings.Trim(strings.TrimSpace(entry), "\"'")

			entryKV := strings.Split(sanitized, "=")
			extraVarsMap[entryKV[0]] = entryKV[1]
		}
	}

	// TODO: add support for setting visibility here OCTRL-178
	// TODO: add support for acquiring bot config here OCTRL-177

	var response *pb.NewEnvironmentReply
	response, err = rpc.NewEnvironment(cxt, &pb.NewEnvironmentRequest{WorkflowTemplate: wfPath, Vars: extraVarsMap}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	_, _ = fmt.Fprintf(o, "new environment created with %s tasks\n", blue(len(tasks)))
	_, _ = fmt.Fprintf(o, "environment id:     %s\n", grey(env.GetId()))
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	_, _ = fmt.Fprintf(o, "root role:          %s\n", env.GetRootRole())

	var (
		defaultsStr = stringMapToString(env.Defaults, "\t")
		varsStr = stringMapToString(env.Vars, "\t")
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

	var response *pb.GetEnvironmentReply
	response, err = rpc.GetEnvironment(cxt, &pb.GetEnvironmentRequest{Id: args[0]}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	env := response.GetEnvironment()
	tasks := env.GetTasks()
	rnString := formatRunNumber(env.GetCurrentRunNumber())

	var (
		defaultsStr = stringMapToString(env.Defaults, "\t")
		varsStr = stringMapToString(env.Vars, "\t")
		userVarsStr = stringMapToString(env.UserVars, "\t")
	)

	_, _ = fmt.Fprintf(o, "environment id:     %s\n", env.GetId())
	_, _ = fmt.Fprintf(o, "created:            %s\n", formatTimestamp(env.GetCreatedWhen()))
	_, _ = fmt.Fprintf(o, "state:              %s\n", colorState(env.GetState()))
	_, _ = fmt.Fprintf(o, "run number:         %s\n", rnString)
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

	keepTasks, err := cmd.Flags().GetBool("keep-tasks")
	if err != nil {
		keepTasks = false
	}

	allowInRunningState, err := cmd.Flags().GetBool("allow-in-running-state")
	if err != nil {
		keepTasks = false
	}

	_, err = rpc.DestroyEnvironment(cxt, &pb.DestroyEnvironmentRequest{Id: envId, KeepTasks: keepTasks, AllowInRunningState: allowInRunningState}, grpc.EmptyCallOption{})
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


func CleanTasks(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) > 0 {
		for _, id := range args {
			if !isValidUUID(id) {
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
			[]string{fmt.Sprintf("task id (%d tasks killed)", len(response.KilledTasks)), "class name", "hostname"},
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
				defaultsStr = stringMapToString(root.Defaults, "\t")
				varsStr = stringMapToString(root.Vars, "\t")
				userVarsStr = stringMapToString(root.UserVars, "\t")
			)

			_, _ = fmt.Fprintf(o, "(%s)\n", yellow(i))
			_, _ = fmt.Fprintf(o, "role path:          %s\n", root.GetFullPath())
			_, _ = fmt.Fprintf(o, "status:             %s\n", root.GetStatus())
			_, _ = fmt.Fprintf(o, "state:              %s\n", root.GetState())
			if len(defaultsStr) != 0 {
				_, _ = fmt.Fprintf(o, "defaults:\n%s\n", defaultsStr)
			}
			if len(varsStr) != 0 {
				_, _ = fmt.Fprintf(o, "variables:\n%s\n", varsStr)
			}
			if len(userVarsStr) != 0 {
				_, _ = fmt.Fprintf(o, "user-provided variables:\n%s\n", userVarsStr)
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

		if allBranches || allTags {
			if revisionPattern != "" {
				fmt.Fprintln(o, "Ignoring `--all-{branches,tags}` flags, as a valid revision has been specified")
				allBranches = false
				allTags = false
			}
		}

	} else 	if len(args) == 1 { // If we have an argument, give priority over the flags
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
		AllBranches: allBranches, AllTags: allTags}, grpc.EmptyCallOption{})
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
			revBranch.AddNode(tmpl.GetTemplate())
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
func AddRepo(cxt context.Context, rpc *coconut.RpcClient, cmd *cobra.Command, args []string, o io.Writer) (err error) {

	name, defaultRevision := "", ""
	if len(args) == 1 {
		name = args[0]
	} else 	if len(args) == 2 {
		name = args[0]
		defaultRevision = args[1]
	} else if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 or 2 args, received %d", len(args)))
		return err
	}

	var response *pb.AddRepoReply
	response, err = rpc.AddRepo(cxt, &pb.AddRepoRequest{Name: name, DefaultRevision: defaultRevision}, grpc.EmptyCallOption{})
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
	fmt.Fprintln(o, "Repository removed succsefully.")
	if newDefaultRepo != "" {
		fmt.Fprintln(o, "New default repo is: " + newDefaultRepo)
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
		fmt.Fprintln(o, "The global default revision has been succesfuly updated to \"" + args[0] + "\".")
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
		fmt.Fprintln(o, "The default revision for this repo has been successfuly updated to \"" + args[1] + "\".")
	} else {
		err := errors.New(fmt.Sprintf("expects 1 or 2 args, received %d", len(args)))
		return err
	}

	return nil
}