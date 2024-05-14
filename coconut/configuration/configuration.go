/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2021 CERN and copyright holders of ALICE O².
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

// Package configuration handles the details of interfacing with
// the O² Configuration store.
package configuration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/apricot"
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/briandowns/spinner"
	"github.com/naoina/toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type CallFunc func(configuration.Service, *cobra.Command, []string, io.Writer) (error, int)

const (
	EC_ZERO             = iota
	EC_INVALID_ARGS     = iota // Provided args by the user are invalid
	EC_INVALID_ARGS_MSG = "component and entry names cannot contain `/` or  `@`"
	EC_CONNECTION_ERROR = iota // Source connection error
	EC_EMPTY_DATA       = iota // Source retrieved empty data
	EC_EMPTY_DATA_MSG   = "no data was found"
	EC_LOGIC_ERROR      = iota // Logic/Output error
)

func WrapCall(call CallFunc) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("config_endpoint")
		log.WithPrefix(cmd.Use).
			WithField("config_endpoint", endpoint).
			Debug("initializing configuration client")

		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		_ = s.Color("yellow")
		s.Suffix = " working..."
		s.Start()

		apricotServiceInstance := apricot.Instance()

		var out strings.Builder

		// redirect stdout to null, the only way to output is
		stdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		err, code := call(apricotServiceInstance, cmd, args, &out)
		os.Stdout = stdout
		s.Stop()
		fmt.Print(out.String())

		if err != nil {
			log.WithPrefix(cmd.Use).
				WithError(err).
				Fatal("command finished with error")
			os.Exit(code)
		}
	}
}

func Dump(svc configuration.Service, cmd *cobra.Command, args []string, o io.Writer) (err error, code int) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return err, EC_INVALID_ARGS
	}
	key := args[0]

	dataJson, err := svc.RawGetRecursive(key)
	if err != nil {
		return err, EC_CONNECTION_ERROR
	}

	var raw map[string]interface{}
	err = json.Unmarshal([]byte(dataJson), &raw)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	var output []byte
	switch strings.ToLower(format) {
	case "json":
		output, err = json.MarshalIndent(raw, "", "    ")
	case "yaml":
		output, err = yaml.Marshal(raw)
	case "toml":
		output, err = toml.Marshal(raw)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return err, EC_LOGIC_ERROR
	}

	_, _ = fmt.Fprintln(o, string(output))

	return nil, EC_ZERO
}

// coconut conf list
func List(svc configuration.Service, cmd *cobra.Command, args []string, o io.Writer) (err error, code int) {
	if len(args) > 1 {
		return errors.New(fmt.Sprintf("command requires maximum 1 arg but received %d", len(args))), EC_INVALID_ARGS
	}

	// we have 0 or 1 args
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	if len(args) == 0 { // 0 args: conf list
		components, _ := svc.ListComponents()
		output, err := formatListOutput(cmd, components)
		if err != nil {
			return err, EC_LOGIC_ERROR
		}
		_, _ = fmt.Fprintln(o, string(output))
		return nil, EC_ZERO
	}

	// 1 arg: conf list <component>
	if !componentcfg.IsInputSingleValidWord(args[0]) {
		return errors.New(EC_INVALID_ARGS_MSG), EC_INVALID_ARGS
	}

	components, _ := svc.ListComponentEntries(&componentcfg.EntriesQuery{
		Component: args[0],
		RunType:   apricotpb.RunType_NULL,
		RoleName:  "",
	})

	output, err := formatListOutput(cmd, components)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}
	_, _ = fmt.Fprintln(o, string(output))
	return nil, EC_ZERO
}

// coconut conf show
func Show(svc configuration.Service, cmd *cobra.Command, args []string, o io.Writer) (err error, code int) {
	// Allowed inputs:
	// # len(args) == 1:
	// coconut conf show component/RUNTYPE/role/entry
	// # len(args) == 2:
	// coconut conf show component entry [--runtype RUNTYPE_EXPR] [--role role_expr]
	// # ↑ if runtype/role is empty, we default to ANY/any
	// #   if it's e.g. SOME_RUNTYPE/some_role we just build the path for now;
	// #   TODO: advanced query expressions come later

	var query *componentcfg.Query
	query, err = queryFromFlags(cmd, args)
	if err != nil {
		return err, EC_INVALID_ARGS
	}
	// At this point we know what to query, thanks to a
	// componentcfg.Query that can be fed to configuration.Instance().
	var (
		cfgPayload string
		simulate   bool
	)
	simulate, err = cmd.Flags().GetBool("simulate")
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	if simulate {
		var extraVars string
		extraVars, err = cmd.Flags().GetString("extra-vars")
		if err != nil {
			return err, EC_INVALID_ARGS
		}

		extraVars = strings.TrimSpace(extraVars)
		if cmd.Flags().Changed("extra-vars") && len(extraVars) == 0 {
			err = errors.New("empty list of extra-vars supplied")
			return err, EC_INVALID_ARGS
		}

		var extraVarsMap map[string]string
		extraVarsMap, err = utils.ParseExtraVars(extraVars)
		if err != nil {
			return err, EC_INVALID_ARGS
		}

		cfgPayload, err = svc.GetAndProcessComponentConfiguration(query, extraVarsMap)
	} else {
		// No template processing simulation requested, getting raw payload
		cfgPayload, err = svc.GetComponentConfiguration(query)
	}
	if err != nil {
		return err, EC_CONNECTION_ERROR
	}
	if cfgPayload == "" {
		return errors.New(EC_EMPTY_DATA_MSG), EC_EMPTY_DATA
	}

	_, _ = fmt.Fprintln(o, cfgPayload)

	return nil, EC_ZERO
}

func Import(svc configuration.Service, cmd *cobra.Command, args []string, o io.Writer) (err error, code int) {
	useNewComponent, err := cmd.Flags().GetBool("new-component")
	if err != nil {
		return err, EC_INVALID_ARGS
	}
	useExtension, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	var pushQuery *componentcfg.Query
	// The last argument is always assumed to be the input file, so we must exclude it
	queryArgs := args[:len(args)-1] // we can do this because it's known that len(args)>0
	pushQuery, err = queryFromFlags(cmd, queryArgs)
	if err != nil {
		return err, EC_INVALID_ARGS
	}
	filePath := args[len(args)-1]

	fileParts := strings.Split(filePath, ".")
	extension := ""
	if len(fileParts) > 1 {
		extension = fileParts[len(fileParts)-1]
	}

	if !isFileExtensionValid(extension) && useExtension == "" {
		return errors.New("supported file extensions: JSON, YAML, INI or TOML." +
			" To force a specific configuration format, see flag --format/-f"), EC_INVALID_ARGS
	} else if useExtension != "" {
		extension = strings.ToUpper(useExtension)
	}

	var payload []byte
	payload, err = getFileContent(filePath)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}

	var existingComponentUpdated, existingEntryUpdated bool
	existingComponentUpdated, existingEntryUpdated, err = svc.ImportComponentConfiguration(pushQuery, string(payload), useNewComponent)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}

	toPrintKey := red(pushQuery.Component) + componentcfg.SEPARATOR +
		blue(pushQuery.RunType) + componentcfg.SEPARATOR +
		red(pushQuery.RoleName) + componentcfg.SEPARATOR +
		blue(pushQuery.EntryKey)

	userMsg := ""
	if !existingComponentUpdated {
		userMsg = "New component created: " + red(pushQuery.Component) + "\n"
	}
	if !existingEntryUpdated {
		userMsg += "New entry created: " + blue(pushQuery.EntryKey) + "\n"
	} else {
		userMsg += "Entry updated: " + blue(pushQuery.EntryKey) + "\n"
	}

	userMsg += "Configuration imported: " + toPrintKey

	_, _ = fmt.Fprintln(o, userMsg)
	return nil, 0
}
