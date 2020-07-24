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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AliceO2Group/Control/common/logger"
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

type ConfigurationCall func(*configuration.ConsulSource, *cobra.Command, []string, io.Writer) (error, int)

const (
	nonZero = iota
	invalidArgs = iota // Provided args by the user are invalid
	invalidArgsErrMsg = "component and entry names cannot contain `/ or  `@`"
	connectionError = iota // Source connection error
	emptyData = iota // Source retrieved empty data
	emptyDataErrMsg = "no data was found"
	logicError = iota // Logic/Output error
)

func WrapCall(call ConfigurationCall) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("config_endpoint")
		log.WithPrefix(cmd.Use).
			WithField("config_endpoint", endpoint).
			Debug("initializing configuration client")

		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		_ = s.Color("yellow")
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

	_, _ = fmt.Fprintln(o, string(output))

	return nil, nonZero
}

func List(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	keyPrefix := componentcfg.ConfigComponentsPath
	useTimestamp := false
	if len(args) > 1 {
		return errors.New(fmt.Sprintf("command requires maximum 1 arg but received %d", len(args))) , invalidArgs
	} else {
		useTimestamp, err = cmd.Flags().GetBool("timestamp")
		if err != nil {
			return err,  invalidArgs
		}
		if len(args) == 1 {
			if !componentcfg.IsInputSingleValidWord(args[0]) {
				return  errors.New(invalidArgsErrMsg), invalidArgs
			} else {
				keyPrefix += args[0] + "/"
			}
		} else if len(args) == 0 && useTimestamp {
			return errors.New("to use flag `-t / --timestamp` please provide component name"), invalidArgs
		}
	}

	keys, err := cfg.GetKeysByPrefix(keyPrefix)
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
	_, _ = fmt.Fprintln(o, string(output))
	return nil, nonZero
}

func Show(cfg *configuration.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var key, component, entry, timestamp string

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("accepts between 0 and 3 arg(s), but received %d", len(args))), invalidArgs
	}

	timestamp, err = cmd.Flags().GetString("timestamp")
	if err != nil {
		return err, invalidArgs
	}

	switch len(args)  {
	case 1:
		if componentcfg.IsInputCompEntryTsValid(args[0] ) {
			if strings.Contains(args[0], "@") {
				if timestamp != "" {
					err = errors.New("flag `-t / --timestamp` must not be provided when using format <component>/<entry>@<timestamp>")
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
			return  errors.New("please provide entry name"), invalidArgs
		}
	case 2:
		if !componentcfg.IsInputSingleValidWord(args[0]) || !componentcfg.IsInputSingleValidWord(args[1]) {
			return errors.New(invalidArgsErrMsg), invalidArgs
		} else {
			component = args[0]
			entry = args[1]
		}
	}

	var cfgPayload string
	if timestamp == "" {
		keyPrefix := componentcfg.ConfigComponentsPath + component + "/" + entry
		keys, err := cfg.GetKeysByPrefix(keyPrefix)
		if err != nil {
			return err, connectionError
		}
		timestamp, err = componentcfg.GetLatestTimestamp(keys,  component , entry)
		if err != nil {
			return err, emptyData
		}
	}
	key = componentcfg.ConfigComponentsPath + component + "/" + entry
	if timestamp != "0" {
		//versioned entry
		key += "/" + timestamp
	}
	cfgPayload, err = cfg.Get(key)
	if err != nil {
		return err, connectionError
	}
	if cfgPayload == ""  {
		return errors.New(emptyDataErrMsg), emptyData
	}

	_, _ = fmt.Fprintln(o, cfgPayload)
	return nil, nonZero
}

func History(cfg *configuration.ConsulSource, _ *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var key, component, entry string

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("accepts between 0 and 3 arg(s), but received %d", len(args))), invalidArgs
	}
	switch len(args) {
	case 1:
		if componentcfg.IsInputSingleValidWord(args[0]) {
			component = args[0]
			entry = ""
		} else if componentcfg.IsInputCompEntryTsValid(args[0]) && !strings.Contains(args[0], "@"){
			splitCom := strings.Split(args[0], "/")
			component = splitCom[0]
			entry = splitCom[1]
		} else {
			return errors.New(invalidArgsErrMsg), invalidArgs
		}
	case 2:
		if componentcfg.IsInputSingleValidWord(args[0]) && componentcfg.IsInputSingleValidWord(args[1]) {
			component = args[0]
			entry = args[1]
		} else {
			return errors.New(invalidArgsErrMsg), invalidArgs
		}
	}

	key = componentcfg.ConfigComponentsPath + component + "/" + entry
	var keys sort.StringSlice
	keys , err = cfg.GetKeysByPrefix(key)
	if err != nil {
		return err, connectionError
	}
	if len(keys) == 0 {
		return errors.New(emptyDataErrMsg), emptyData
	} else {
		if entry != "" {
			sort.Sort(sort.Reverse(keys))
			drawTableHistoryConfigs([]string{}, keys, 0, o)
		} else {
			maxLen := getMaxLenOfKey(keys)
			var currentKeys sort.StringSlice
			_, entry, _ := componentcfg.GetComponentEntryTimestampFromConsul(keys[0])

			for _, value := range keys {
				_, currentEntry, _ := componentcfg.GetComponentEntryTimestampFromConsul(value)
				if currentEntry == entry {
					currentKeys = append(currentKeys, value)
				} else {
					_, _ = fmt.Fprintln(o, "- "+entry)
					sort.Sort(sort.Reverse(currentKeys)) //sort in reverse of timestamps
					drawTableHistoryConfigs([]string{}, currentKeys,maxLen, o)
					currentKeys = []string{value}
					entry = currentEntry
				}
			}
			_, _ = fmt.Fprintln(o, "- "+entry)
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
	useNoVersion, err := cmd.Flags().GetBool("no-versioning")
	if err != nil {
		return err, invalidArgs
	}

	// Parse and Format input arguments
	var component, entry, filePath string
	if len(args) < 2 ||  len(args) > 3 {
		return errors.New(fmt.Sprintf("accepts 2 or 3 args but received %d", len(args))), invalidArgs
	} else {
		switch len(args) {
		case 2:
			component, entry, err = getComponentEntryFromUserInput(args[0])
			filePath = args[1]
		case 3:
			if !componentcfg.IsInputSingleValidWord(args[0]) || !componentcfg.IsInputSingleValidWord(args[1])  {
				err = errors.New(invalidArgsErrMsg)
			} else {
				component = args[0]
				entry = args[1]
				filePath = args[2]
			}
		}
		if err != nil {
			return err, invalidArgs
		}
	}

	fileParts := strings.Split(filePath, ".")
	extension := ""
	if len(fileParts) > 1 {
		extension = fileParts[len(fileParts)-1]
	}

	if !isFileExtensionValid(extension) &&  useExtension == "" {
		return errors.New("supported file extensions: JSON, YAML, INI or TOML." +
			" To force a specific configuration format, see flag --format/-f" ), invalidArgs
	} else if useExtension != ""  {
		extension = strings.ToUpper(useExtension)
	}

	keys, err := cfg.GetKeysByPrefix("")
	if err != nil {
		return  err, connectionError
	}

	components := componentcfg.GetComponentsMapFromKeysList(keys)
	componentExist := components[component]
	if !componentExist &&  !useNewComponent {
		componentMsg := ""
		for key := range components {
			componentMsg += "\n- " + key
		}
		return errors.New("component " + component + " does not exist. " +
			"Available components in configuration database:" +  componentMsg +
			"\nTo create a new component, see flag --new-component/-n" ),
			invalidArgs
	}
	if componentExist && useNewComponent {
		return errors.New("invalid use of flag --new-component/-n: component " + red(component) + " already exists"), invalidArgs
	}

	entryExists := false
	if !useNewComponent {
		entriesMap := componentcfg.GetEntriesMapOfComponentFromKeysList(component, keys)
		entryExists = entriesMap[entry]
	}

	fileContent, err := getFileContent(filePath)
	if err != nil {
		return err, logicError
	}

	// Temporary workaround to allow no-versioning
	latestTimestamp, err := componentcfg.GetLatestTimestamp(keys, component, entry)
	if err != nil {
		return err, invalidArgs
	}

	if entryExists {
		if (latestTimestamp != "0" && latestTimestamp != "") && useNoVersion {
			// If a timestamp already exists in the entry specified by the user, than it cannot be used
			return errors.New("Specified entry: '" + entry + "' already contains versioned items. Please " +
				"specify a different entry name"), invalidArgs
		}
		if (latestTimestamp == "0" || latestTimestamp == "") && !useNoVersion {
			// If a timestamp does not exist for specified entry but user wants versioning than an error is thrown
			return errors.New("Specified entry: '" + entry + "' already contains un-versioned items. Please " +
				"specify a different entry name"), invalidArgs
		}
	}

	timestamp := time.Now().Unix()
	fullKey := componentcfg.ConfigComponentsPath + component + "/" + entry
	toPrintKey :=  red(component) + "/" + blue(entry)

	if !useNoVersion {
		fullKey += "/" + strconv.FormatInt(timestamp, 10)
		toPrintKey += "@" + strconv.FormatInt(timestamp, 10)
	}

	err = cfg.Put(fullKey, string(fileContent))
	if err != nil {
		return
	}

	userMsg := ""
	if !componentExist {
		userMsg = "New component created: " + red(component) + "\n"
	}
	if !entryExists {
		userMsg += "New entry created: " + blue(entry) + "\n"
	} else {
		userMsg += "Entry updated: " + blue(entry) +  "\n"
	}

	userMsg += "Configuration imported: " + toPrintKey

	_, _ = fmt.Fprintln(o, userMsg)
	return nil, 0
}

