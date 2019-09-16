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

// Package configuration handles the details of interfacing with
// the O² Configuration store.
package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/briandowns/spinner"
	"time"
	"io"
	"github.com/sirupsen/logrus"
	"os"
	"github.com/AliceO2Group/Control/common/logger"
	"fmt"
	"strings"
	"github.com/AliceO2Group/Control/configuration"
	"errors"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"github.com/naoina/toml"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type ConfigurationCall func(*configuration.ConsulSource, *cobra.Command, []string, io.Writer) (error, int)

var componentsPath = "o2/components/"
// code = 1 - Provided args by the user are invalid
// code = 2 - Source connection error
// code = 3 - Source retrieved empty data
// code = 4 - Logic/Output error

func WrapCall(call ConfigurationCall) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("config_endpoint")
		log.WithPrefix(cmd.Use).
			WithField("config_endpoint", endpoint).
			Debug("initializing configuration client")

		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Color("yellow")
		s.Suffix = " working..."
		s.Start()

		cfg, err := configuration.NewConsulSource(strings.TrimPrefix(endpoint, "consul://"))
		if err != nil {
			var fields logrus.Fields
			if logrus.GetLevel() == logrus.DebugLevel {
				fields = logrus.Fields{"error": err}
			}
			log.WithPrefix(cmd.Use).
				WithFields(fields).
				Fatal("cannot query endpoint")
			os.Exit(2)
		}

		var out strings.Builder

		// redirect stdout to null, the only way to output is
		stdout := os.Stdout
		os.Stdout,_ = os.Open(os.DevNull)
		err, code := call(cfg, cmd, args, &out)
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

func Dump(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer) (err error, code int) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return err, 1
	}
	key := args[0]

	data, err := cfg.GetRecursive(key)
	if err != nil {
		return err, 2
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, 1
	}

	var output []byte
	switch strings.ToLower(format) {
	case "json":
		output, err = json.MarshalIndent(data, "", "    ")
	case "yaml":
		output, err = yaml.Marshal(data)
	case "toml":
		output, err = toml.Marshal(data)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return err, 4
	}

	fmt.Fprintln(o, string(output))

	return nil, 0
}

func List(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	keyPrefix := componentsPath
	useTimestamp := false
	if len(args) > 1 {
		err = errors.New(fmt.Sprintf("Command requires maximum 1 arg but received %d", len(args)))
		return err , 1
	} else {
		useTimestamp, err = cmd.Flags().GetBool("timestamp")
		if err != nil {
			err = errors.New(fmt.Sprintf("Flag `-t / --timestamp` could not be identified"))
			return err, 1
		}
		if len(args) == 1 {
			if !isInputNameValid(args[0]) {
				err = errors.New(fmt.Sprintf("Requested component name cannot contain character `/` or `@`"))
				return err, 1
			} else {
				keyPrefix += args[0] + "/"
			}
		} else if len(args) == 0 && useTimestamp {
			err = errors.New(fmt.Sprintf("To use flag `-t / --timestamp` please provide component name"))
			return err, 1
		}
	}

	keys, err := cfg.GetKeysByPrefix(keyPrefix, "")
	if err != nil {
		err = errors.New(fmt.Sprintf("Could not query ConsulSource"))
		return err, 2
	}

	// keys is empty? for a certain component? throw error maybe?

	var components []string
	componentsSet := make(map[string]string)

	for _, key := range keys {
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, "/")
		componentTimestamp := componentParts[len(componentParts) - 1]
		if useTimestamp {
			componentsFullName = strings.TrimSuffix(componentsFullName, "/" +componentTimestamp)
		} else {
			componentsFullName = componentParts[0]
		}

		if strings.Compare(componentsSet[componentsFullName], componentTimestamp) < 0{
			componentsSet[componentsFullName] = componentTimestamp
		}
	}

	for key,value := range componentsSet {
		if useTimestamp {
			components = append(components, key+"@"+value)
		} else {
			components = append(components, key)
		}
	}

	output, err := formatListOutput(cmd, components)
	if err != nil {
		return err, 4
	}
	fmt.Fprintln(o, string(output))
	return nil, 0
}

func Show(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var key, component, entry, timestamp string
	configMap := make(map[string]string)

	if len(args) < 1 ||  len(args) > 2 {
		err = errors.New(fmt.Sprintf(" accepts between 0 and 3 arg(s), but received %d", len(args)))
		return err, 1
	}

	timestamp, err = cmd.Flags().GetString("timestamp")
	if err != nil {
		err = errors.New(fmt.Sprintf("Flag `-t / --timestamp` could not be provided"))
		return err, 1
	}

	switch len(args)  {
	case 1:
		if strings.Contains(args[0], "@")  && strings.Contains(args[0], "/"){
			if timestamp != "" {
				err = errors.New(fmt.Sprintf("Flag `-t / --timestamp` must not be provided when using format `component/entry@timestamp`"))
				return err, 1
			}
			// coconut conf show component/entry@timestamp
			arg := strings.Replace(args[0], "@", "/", 1)
			params := strings.Split(arg, "/")
			component = params[0]
			entry = params[1]
			timestamp = params[2]
		} else if strings.Contains(args[0], "/") {
			// assumes component/entry
			params := strings.Split(args[0], "/")
			component = params[0]
			entry = params[1]
		} else {
			// coconut conf show  component / coconut conf show component@timestamp
			err = errors.New(fmt.Sprintf("Please provide entry name"))
			return err, 1
		}
	case 2:
		if !isInputNameValid(args[0]) || !isInputNameValid(args[1]) {
			err = errors.New(fmt.Sprintf("Component or Entry name provided are not valid"))
			return err, 1
		} else {
			component = args[0]
			entry = args[1]
		}
	}

	var configuration string
	if timestamp == "" {
		timestamp, err, code = getLatestTimestamp(cfg,  component , entry)
		if err != nil {
			return err, code
		}
	}
	key = componentsPath + component + "/" + entry + "/" + timestamp
	configuration, err = cfg.Get(key)
	if err != nil {
		return err, 2
	}
	if configuration == ""  {
		err = errors.New(fmt.Sprintf("Requsted component and entry could not be found"))
		return err, 3
	}
	configMap[timestamp] = configuration
	output, err, code := formatConfigOutput(cmd, configMap)
	if err != nil {
		return err, code
	}

	fmt.Fprintln(o, string(output))
	return nil, 0
}

func formatListOutput( cmd *cobra.Command, output []string)(parsedOutput []byte, err error) {
	format, err := cmd.Flags().GetString("output")
	if err != nil {
		return
	}

	switch strings.ToLower(format) {
		case "json":
			parsedOutput, err = json.MarshalIndent(output, "", "    ")
		case "yaml":
			parsedOutput, err = yaml.Marshal(output)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return
	}
	return parsedOutput, nil
}

func formatConfigOutput( cmd *cobra.Command, output map[string]string)(parsedOutput []byte, err error, code int) {
	format, err := cmd.Flags().GetString("output")
	if err != nil {
		code  = 1
		return
	}

	switch strings.ToLower(format) {
	case "json":
		parsedOutput, err = json.MarshalIndent(output, "", "    ")
	case "yaml":
		parsedOutput, err = yaml.Marshal(output)
	case "toml":
		parsedOutput, err = toml.Marshal(output)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return
	}
	return parsedOutput, nil, 0
}

func isInputNameValid(input string)(valid bool) {
	if strings.Contains(input, "/") || strings.Contains(input, "@") {
		return false
	} else {
		return true
	}
}

func getLatestTimestamp(cfg configuration.Source, component string, entry string)(timestamp string, err error, code int) {
	keyPrefix := componentsPath + component + "/" + entry
	keys, err := cfg.GetKeysByPrefix(keyPrefix, "")
	if err != nil {
		err = errors.New(fmt.Sprintf("Could not query ConsulSource"))
		return "", err, 2
	}
	if len(keys) == 0 {
		err = errors.New(fmt.Sprintf("No keys found"))
		return "", err, 3
	}

	var maxTimeStamp string
	for _, key := range keys {
		componentsFullName := strings.TrimPrefix(key, keyPrefix + "/")
		if strings.Compare(componentsFullName, maxTimeStamp) > 0 {
			maxTimeStamp = componentsFullName
		}
	}
	return maxTimeStamp, nil, 0
}