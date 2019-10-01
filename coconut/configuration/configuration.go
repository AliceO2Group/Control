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
	"strconv"
	"time"
	"io"
	"github.com/sirupsen/logrus"
	"os"
	"sort"
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

const  (
	nonZero = iota
	invalidArgs = iota // Provided args by the user are invalid
	invalidArgsErrMsg = "Component and Entry names cannot contain `/ or  `@`"
	connectionError = iota // Source connection error
	consulConnectionErrMsg = "Could not query ConsulSource"
	emptyData = iota // Source retrieved empty data
	emptyDataErrMsg = "No data was found"
	logicError = iota // Logic/Output error
	componentsPath = "o2/components/"
)

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
			os.Exit(connectionError)
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
				Fatal( "command finished with error")
			os.Exit(code)
		}
	}
}

func Dump(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer) (err error,  code int) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return err, invalidArgs
	}
	key := args[0]

	data, err := cfg.GetRecursive(key)
	if err != nil {
		return err, connectionError
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, invalidArgs
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
		return err, logicError
	}

	fmt.Fprintln(o, string(output))

	return nil, nonZero
}

func List(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	keyPrefix := componentsPath
	useTimestamp := false
	if len(args) > 1 {
		return errors.New(fmt.Sprintf("Command requires maximum 1 arg but received %d", len(args))) , invalidArgs
	} else {
		useTimestamp, err = cmd.Flags().GetBool("timestamp")
		if err != nil {
			return err,  invalidArgs
		}
		if len(args) == 1 {
			if !isInputSingleValidWord(args[0]) {
				return  errors.New(fmt.Sprintf(invalidArgsErrMsg)), invalidArgs
			} else {
				keyPrefix += args[0] + "/"
			}
		} else if len(args) == 0 && useTimestamp {
			return errors.New(fmt.Sprintf("To use flag `-t / --timestamp` please provide component name")), invalidArgs
		}
	}

	keys, err := cfg.GetKeysByPrefix(keyPrefix, "")
	if err != nil {
		return  err, connectionError
	}

	components, err, code := getListOfComponentsAndOrWithTimestamps(keys, keyPrefix, useTimestamp)
	if err != nil {
		return err, code
	}

	output, err := formatListOutput(cmd, components)
	if err != nil {
		return err, logicError
	}
	fmt.Fprintln(o, string(output))
	return nil, nonZero
}

func Show(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var key, component, entry, timestamp string

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("Accepts between 0 and 3 arg(s), but received %d", len(args))), invalidArgs
	}

	timestamp, err = cmd.Flags().GetString("timestamp")
	if err != nil {
		return err, invalidArgs
	}

	switch len(args)  {
	case 1:
		if isInputCompEntryTsValid(args[0] ) {
			if strings.Contains(args[0], "@") {
				if timestamp != "" {
					err = errors.New(fmt.Sprintf("Flag `-t / --timestamp` must not be provided when using format `component/entry@timestamp`"))
					return err, invalidArgs
				}
				// coconut conf show component/entry@timestamp
				arg := strings.Replace(args[0], "@", "/", 1)
				params := strings.Split(arg, "/")
				component = params[0]
				entry = params[1]
				timestamp = params[2]
			} else if strings.Contains(args[0], "/") {
				// coconut conf show component/entry
				params := strings.Split(args[0], "/")
				component = params[0]
				entry = params[1]
			}
		} else {
			// coconut conf show  component || coconut conf show component@timestamp
			return  errors.New(fmt.Sprintf("Please provide entry name")), invalidArgs
		}
	case 2:
		if !isInputSingleValidWord(args[0]) || !isInputSingleValidWord(args[1]) {
			return errors.New(fmt.Sprintf(invalidArgsErrMsg)), invalidArgs
		} else {
			component = args[0]
			entry = args[1]
		}
	}

	var configuration string
	if timestamp == "" {
		keyPrefix := componentsPath + component + "/" + entry
		keys, err := cfg.GetKeysByPrefix(keyPrefix, "")
		if err != nil {
			return err, connectionError
		}
		timestamp, err, code = getLatestTimestamp(keys,  component , entry)
		if err != nil {
			return err, code
		}
	}
	key = componentsPath + component + "/" + entry + "/" + timestamp
	configuration, err = cfg.Get(key)
	if err != nil {
		return err, connectionError
	}
	if configuration == ""  {
		return errors.New(fmt.Sprintf(emptyDataErrMsg)), emptyData
	}

	fmt.Fprintln(o, configuration)
	return nil, nonZero
}

func History(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var key, component, entry string

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("Accepts between 0 and 3 arg(s), but received %d", len(args))), invalidArgs
	}
	switch len(args) {
	case 1:
		if isInputSingleValidWord(args[0]) {
			component = args[0]
			entry = ""
		} else if isInputCompEntryTsValid(args[0]) && !strings.Contains(args[0], "@"){
			splitCom := strings.Split(args[0], "/")
			component = splitCom[0]
			entry = splitCom[1]
		} else {
			return errors.New(fmt.Sprintf(invalidArgsErrMsg)), invalidArgs
		}
	case 2:
		if isInputSingleValidWord(args[0]) && isInputSingleValidWord(args[1]) {
			component = args[0]
			entry = args[1]
		} else {
			return errors.New(fmt.Sprintf(invalidArgsErrMsg)), invalidArgs
		}
	}

	key = componentsPath + component + "/" + entry
	var keys sort.StringSlice
	keys , err = cfg.GetKeysByPrefix(key, "")
	if err != nil {
		return err, connectionError
	}
	if len(keys) == 0 {
		return errors.New(fmt.Sprintf(emptyDataErrMsg)), emptyData
	} else {
		if entry != "" {
			sort.Sort(sort.Reverse(keys))
			drawTableHistoryConfigs([]string{}, keys, 0, o)
		} else {
			maxLen := getMaxLenOfKey(keys)
			var currentKeys sort.StringSlice
			_, entry, _ := getComponentEntryTimestampFromConsul(keys[0])

			for _, value := range keys {
				_, currentEntry, _ := getComponentEntryTimestampFromConsul(value)
				if currentEntry == entry {
					currentKeys = append(currentKeys, value)
				} else {
					fmt.Fprintln(o, "- " + entry)
					sort.Sort(sort.Reverse(currentKeys)) //sort in reverse of timestamps
					drawTableHistoryConfigs([]string{}, currentKeys,maxLen, o)
					currentKeys = []string{value}
					entry = currentEntry
				}
			}
			fmt.Fprintln(o, "- " + entry)
			drawTableHistoryConfigs([]string{}, currentKeys,maxLen, o)
		}
	}
	return nil, 0
}

func Import(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	useNewComponent, err := cmd.Flags().GetBool("new-component")
	if err != nil {
		return err, invalidArgs
	}
	useExtension, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, invalidArgs
	}
	if len(args) != 3 {
		return errors.New(fmt.Sprintf("Accepts exactly 3 args but received %d", len(args))), invalidArgs
	}

	if !isInputSingleValidWord(args[0]) || !isInputSingleValidWord(args[1]) && args[2] != "" {
		return errors.New(fmt.Sprintf(invalidArgsErrMsg)), invalidArgs
	}

	component, entry, filePath := args[0], args[1], args[2]

	fileParts := strings.Split(filePath, ".")
	extension := ""
	if len(fileParts) > 1 {
		extension = fileParts[len(fileParts)-1]
	}

	if !isFileExtensionValid(extension) &&  useExtension == "" {
		return errors.New(fmt.Sprintf("Extension of the file should be: JSON, YAML, INI or TOML  or for a different extension " +
			"please use flag '-f/--format' and specify the extension.", )), invalidArgs
	} else if useExtension != ""  {
		extension = strings.ToUpper(useExtension)
	}

	keys, err := cfg.GetKeysByPrefix("", "")
	if err != nil {
		return  err, connectionError
	}

	components := getComponentsMapFromKeysList(keys)
	componentExist := components[component]
	if !componentExist &&  !useNewComponent {
		componentMsg := ""
		for key, _ := range components {
			componentMsg += "\n-" + key
		}
		return errors.New(fmt.Sprintf("Component `" + component + "` does not exist. " +
			"Please check through the already existing components:" +
			componentMsg + "\nIf you wish to add a new component, please use flag `-n/-new-component` to create a new component\n" )),
			invalidArgs
	}
	if componentExist && useNewComponent {
		return errors.New(fmt.Sprintf("Component `" + component + "` already exists, thus flag  `-n/-new-component` cannot be used" )), invalidArgs
	}

	entryExists := false
	if useNewComponent {
		entryExists = false
	} else {
		entriesMap := getEntriesMapOfComponentFromKeysList(component, keys)
		entryExists = entriesMap[entry]
	}

	fileContent, err := getFileContent(filePath)
	if err != nil {
		return err, logicError
	}

	timestamp := time.Now().Unix()
	key := componentsPath + component + "/" + entry + "/" + strconv.FormatInt(timestamp, 10)
	err = cfg.Put(key, string(fileContent))
	if err != nil {
		return
	}

	userMsg := ""
	if !componentExist {
		userMsg = "A new component has been created: " + red(component) + "\n"
	} else {
		userMsg += "Component " + red(component) + " has been updated" +  "\n"
	}
	if !entryExists {
		userMsg += "A new entry has been created: " + blue(entry) + "\n"
	} else {
		userMsg += "Entry " + blue(entry) + " has been updated" +  "\n"
	}
	fullKey :=  red(component) + "/" + blue(entry) + "@" + strconv.FormatInt(timestamp, 10)
	userMsg += "The following configuration key has been added: " + fullKey

	fmt.Fprintln(o, userMsg)
	return nil, 0
}

