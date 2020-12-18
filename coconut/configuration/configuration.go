/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration/cfgbackend"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/configuration/the"
	"github.com/briandowns/spinner"
	"github.com/naoina/toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type ConfigurationCall func(*cfgbackend.ConsulSource, *cobra.Command, []string, io.Writer) (error, int)

const (
	EC_ZERO             = iota
	EC_INVALID_ARGS     = iota // Provided args by the user are invalid
	EC_INVALID_ARGS_MSG = "component and entry names cannot contain `/` or  `@`"
	EC_CONNECTION_ERROR = iota // Source connection error
	EC_EMPTY_DATA       = iota // Source retrieved empty data
	EC_EMPTY_DATA_MSG   = "no data was found"
	EC_LOGIC_ERROR      = iota // Logic/Output error
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

		cfg, err := cfgbackend.NewConsulSource(strings.TrimPrefix(endpoint, "consul://"))
		if err != nil {
			var fields logrus.Fields
			if logrus.GetLevel() == logrus.DebugLevel {
				fields = logrus.Fields{"error": err}
			}
			log.WithPrefix(cmd.Use).
				WithFields(fields).
				Fatal("cannot query endpoint")
			os.Exit(EC_CONNECTION_ERROR)
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

func Dump(cfg *cfgbackend.ConsulSource, cmd *cobra.Command, args []string, o io.Writer) (err error,  code int) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return err, EC_INVALID_ARGS
	}
	key := args[0]

	data, err := cfg.GetRecursive(key)
	if err != nil {
		return err, EC_CONNECTION_ERROR
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, EC_INVALID_ARGS
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
		return err, EC_LOGIC_ERROR
	}

	_, _ = fmt.Fprintln(o, string(output))

	return nil, EC_ZERO
}

// coconut conf list
func List(cfg *cfgbackend.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	keyPrefix := componentcfg.ConfigComponentsPath
	useTimestamp := false
	if len(args) > 1 {
		return errors.New(fmt.Sprintf("command requires maximum 1 arg but received %d", len(args))) , EC_INVALID_ARGS
	} else {
		// case with 0 or 1 args (+ optionally timestamp)
		useTimestamp, err = cmd.Flags().GetBool("timestamp")
		if err != nil {
			return err, EC_INVALID_ARGS
		}
		if len(args) == 1 {
			if !componentcfg.IsInputSingleValidWord(args[0]) {
				return  errors.New(EC_INVALID_ARGS_MSG), EC_INVALID_ARGS
			} else {
				keyPrefix += args[0] + "/"
			}
		} else if len(args) == 0 && useTimestamp {
			return errors.New("to use flag `-t / --timestamp` please provide component name"), EC_INVALID_ARGS
		}
	}

	keys, err := cfg.GetKeysByPrefix(keyPrefix)
	if err != nil {
		return  err, EC_CONNECTION_ERROR
	}
	fmt.Fprintf(o, "keys:\n%s", strings.Join(keys, "\n"))

	components, err, code := getListOfComponentsAndOrWithTimestamps(keys, keyPrefix, useTimestamp)
	if err != nil {
		return err, code
	}
	fmt.Fprintf(o, "\ncomponents:\n%s", strings.Join(components, "\n"))

	output, err := formatListOutput(cmd, components)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}
	_, _ = fmt.Fprintln(o, string(output))
	return nil, EC_ZERO
}

func Show(cfg *cfgbackend.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	var timestamp string
	var query = &componentcfg.Query{}

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("accepts 1 or 2 arg(s), but received %d", len(args))), EC_INVALID_ARGS
	}

	timestamp, err = cmd.Flags().GetString("timestamp")
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	switch len(args) {
	case 1:	// coconut conf show component/RUNTYPE/role/entry[@timestamp]
		if componentcfg.IsInputCompEntryTsValid(args[0] ) {
			if strings.Contains(args[0], "@") {
				// coconut conf show c/R/r/e@timestamp
				if timestamp != "" {
					err = errors.New("flag `-t / --timestamp` must not be provided when using format <component>/<runtype>/<role>/<entry>@<timestamp>")
					return err, EC_INVALID_ARGS
				}
				query, err = componentcfg.NewQuery(args[0])
				if err != nil {
					return err, EC_INVALID_ARGS
				}
				timestamp = query.Timestamp
			} else if strings.Contains(args[0], "/") {
				// coconut conf show c/R/r/e    # no timestamp
				query, err = componentcfg.NewQuery(args[0])
				if err != nil {
					return err, EC_INVALID_ARGS
				}
				if timestamp == "" {
					timestamp = query.Timestamp
				}
			}
		} else {
			// coconut conf show  component || coconut conf show component@timestamp
			return  errors.New("please provide entry name"), EC_INVALID_ARGS
		}
	case 2:	// coconut conf show component entry
		if !componentcfg.IsInputSingleValidWord(args[0]) || !componentcfg.IsInputSingleValidWord(args[1]) {
			return errors.New(EC_INVALID_ARGS_MSG), EC_INVALID_ARGS
		} else {
			query.Component = args[0]
			query.EntryKey = args[1]
			query.Flavor = apricotpb.RunType_ANY
			query.Rolename = "any"
		}
	}

	fullKeyToQuery := query.AbsoluteWithoutTimestamp()

	if timestamp == "" {
		// No timestamp was passed, either via -t or @
		// We need to ascertain whether the required entry is versioned
		// or unversioned.
		keyPrefix := query.AbsoluteWithoutTimestamp()
		if cfg.IsDir(keyPrefix) {
			// The requested path is a Consul folder, so we should
			// look inside to find the latest timestamp.
			keys, err := cfg.GetKeysByPrefix(keyPrefix)
			if err != nil {
				return err, EC_CONNECTION_ERROR
			}
			timestamp, err = componentcfg.GetLatestTimestamp(keys, query)
			if err != nil {
				return err, EC_EMPTY_DATA
			}
			fullKeyToQuery += componentcfg.SEPARATOR + timestamp
		}
		// Otherwise, the requested path is not a Consul folder, so it must
		// be a plain unversioned entry.
		// Therefore the fullKeyToQuery is ok and there's nothing to do.
	} else {
		// A timestamp was passed, either via -t or @
		// We can safely append it to the full query path.
		fullKeyToQuery += componentcfg.SEPARATOR + timestamp
	}
	query.Timestamp = timestamp

	// At this point we know what to query, either fullKeyToQuery
	// for a raw configuration.ConsulSource query, or a
	// componentcfg.Query that can be fed to the.ConfSvc().
	var(
		cfgPayload string
		simulate bool
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

		fmt.Fprintf(o,"%s", query.Path())
		cfgPayload, err = the.ConfSvc().GetAndProcessComponentConfiguration(query, extraVarsMap)
		if err != nil {
			return err, EC_CONNECTION_ERROR
		}
	} else {
		// No template processing simulation requested, getting raw payload
		cfgPayload, err = cfg.Get(fullKeyToQuery)
		if err != nil {
			return err, EC_CONNECTION_ERROR
		}
		if cfgPayload == ""  {
			return errors.New(EC_EMPTY_DATA_MSG), EC_EMPTY_DATA
		}
	}
	_, _ = fmt.Fprintln(o, cfgPayload)

	return nil, EC_ZERO
}

func History(cfg *cfgbackend.ConsulSource, _ *cobra.Command, args []string, o io.Writer)(err error, code int) {
	p := &componentcfg.Query{}

	if len(args) < 1 ||  len(args) > 2 {
		return errors.New(fmt.Sprintf("accepts 1 or 2 arg(s), but received %d", len(args))), EC_INVALID_ARGS
	}
	switch len(args) {
	case 1:
		if componentcfg.IsInputSingleValidWord(args[0]) {
			p.Component = args[0]
		} else if componentcfg.IsInputCompEntryTsValid(args[0]) && !strings.Contains(args[0], "@"){
			p, err = componentcfg.NewQuery(args[0])
			if err != nil {
				return err, EC_INVALID_ARGS
			}
		} else {
			return errors.New(EC_INVALID_ARGS_MSG), EC_INVALID_ARGS
		}
	case 2:
		if componentcfg.IsInputSingleValidWord(args[0]) && componentcfg.IsInputSingleValidWord(args[1]) {
			p.Component = args[0]
			p.EntryKey = args[1]
			p.Flavor = apricotpb.RunType_ANY
			p.Rolename = "any"
		} else {
			return errors.New(EC_INVALID_ARGS_MSG), EC_INVALID_ARGS
		}
	}

	fullKeyToQuery := p.AbsoluteWithoutTimestamp()
	var keys sort.StringSlice
	keys , err = cfg.GetKeysByPrefix(fullKeyToQuery)
	if err != nil {
		return err, EC_CONNECTION_ERROR
	}
	if len(keys) == 0 {
		return errors.New(EC_EMPTY_DATA_MSG), EC_EMPTY_DATA
	} else {
		if p.EntryKey != "" {
			sort.Sort(sort.Reverse(keys))
			drawTableHistoryConfigs([]string{}, keys, 0, o)
		} else {
			maxLen := getMaxLenOfKey(keys)
			var currentKeys sort.StringSlice

			entry := p.EntryKey

			for _, value := range keys {
				var thisPath *componentcfg.Query
				thisPath, err = componentcfg.NewQuery(value)
				if err != nil {
					continue
				}

				if thisPath.EntryKey == entry {
					currentKeys = append(currentKeys, value)
				} else {
					_, _ = fmt.Fprintln(o, "- "+entry)
					sort.Sort(sort.Reverse(currentKeys)) //sort in reverse of timestamps
					drawTableHistoryConfigs([]string{}, currentKeys, maxLen, o)
					currentKeys = []string{value}
					entry = thisPath.EntryKey
				}
			}
			_, _ = fmt.Fprintln(o, "- "+entry)
			drawTableHistoryConfigs([]string{}, currentKeys, maxLen, o)
		}
	}
	return nil, 0
}

func Import(cfg *cfgbackend.ConsulSource, cmd *cobra.Command, args []string, o io.Writer)(err error, code int) {
	useNewComponent, err := cmd.Flags().GetBool("new-component")
	if err != nil {
		return err, EC_INVALID_ARGS
	}
	useExtension, err := cmd.Flags().GetString("format")
	if err != nil {
		return err, EC_INVALID_ARGS
	}
	useNoVersion, err := cmd.Flags().GetBool("no-versioning")
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	// Parse and Format input arguments
	var filePath string
	var p = &componentcfg.Query{}

	if len(args) < 2 || len(args) > 3 {
		return errors.New(fmt.Sprintf("accepts 2 or 3 args but received %d", len(args))), EC_INVALID_ARGS
	} else {
		switch len(args) {
		case 2: // coconut conf import component/RUNTYPE/role/entry filepath
			p, err = componentcfg.NewQuery(args[0])
			if err != nil {
				return err, EC_INVALID_ARGS
			}
			filePath = args[1]
		case 3: // coconut conf import component entry filepath  # ANY/any assumed
			if !componentcfg.IsInputSingleValidWord(args[0]) || !componentcfg.IsInputSingleValidWord(args[1])  {
				err = errors.New(EC_INVALID_ARGS_MSG)
			} else {
				p.Component = args[0]
				p.EntryKey = args[1]
				p.Rolename = "any"
				p.Flavor = apricotpb.RunType_ANY
				filePath = args[2]
			}
		}
		if err != nil {
			return err, EC_INVALID_ARGS
		}
	}

	fileParts := strings.Split(filePath, ".")
	extension := ""
	if len(fileParts) > 1 {
		extension = fileParts[len(fileParts)-1]
	}

	if !isFileExtensionValid(extension) &&  useExtension == "" {
		return errors.New("supported file extensions: JSON, YAML, INI or TOML." +
			" To force a specific configuration format, see flag --format/-f" ), EC_INVALID_ARGS
	} else if useExtension != ""  {
		extension = strings.ToUpper(useExtension)
	}

	keys, err := cfg.GetKeysByPrefix("")
	if err != nil {
		return  err, EC_CONNECTION_ERROR
	}

	components := componentcfg.GetComponentsMapFromKeysList(keys)

	componentExist := components[p.Component]
	if !componentExist && !useNewComponent {
		componentMsg := ""
		for key := range components {
			componentMsg += "\n- " + key
		}
		return errors.New("component " + p.Component + " does not exist. " +
			"Available components in configuration database:" +  componentMsg +
			"\nTo create a new component, see flag --new-component/-n" ),
			EC_INVALID_ARGS
	}
	if componentExist && useNewComponent {
		return errors.New("invalid use of flag --new-component/-n: component " + red(p.Component) + " already exists"), EC_INVALID_ARGS
	}

	entryExists := false
	if !useNewComponent {
		entriesMap := componentcfg.GetEntriesMapOfComponentFromKeysList(p.Component, p.Flavor, p.Rolename, keys)
		entryExists = entriesMap[p.EntryKey]
	}

	fileContent, err := getFileContent(filePath)
	if err != nil {
		return err, EC_LOGIC_ERROR
	}

	// Temporary workaround to allow no-versioning
	latestTimestamp, err := componentcfg.GetLatestTimestamp(keys, p)
	if err != nil {
		return err, EC_INVALID_ARGS
	}

	if entryExists {
		if (latestTimestamp != "0" && latestTimestamp != "") && useNoVersion {
			// If a timestamp already exists in the entry specified by the user, than it cannot be used
			return errors.New("Specified entry: '" + p.EntryKey + "' already contains versioned items. Please " +
				"specify a different entry name"), EC_INVALID_ARGS
		}
		if (latestTimestamp == "0" || latestTimestamp == "") && !useNoVersion {
			// If a timestamp does not exist for specified entry but user wants versioning than an error is thrown
			return errors.New("Specified entry: '" + p.EntryKey + "' already contains un-versioned items. Please " +
				"specify a different entry name"), EC_INVALID_ARGS
		}
	}

	timestamp := time.Now().Unix()
	fullKey := p.AbsoluteWithoutTimestamp()
	toPrintKey :=  red(p.Component) + componentcfg.SEPARATOR +
		blue(p.Flavor) + componentcfg.SEPARATOR +
		red(p.Rolename) + componentcfg.SEPARATOR +
		blue(p.EntryKey)

	if !useNoVersion {
		fullKey += componentcfg.SEPARATOR + strconv.FormatInt(timestamp, 10)
		toPrintKey += "@" + strconv.FormatInt(timestamp, 10)
	}

	err = cfg.Put(fullKey, string(fileContent))
	if err != nil {
		return
	}

	userMsg := ""
	if !componentExist {
		userMsg = "New component created: " + red(p.Component) + "\n"
	}
	if !entryExists {
		userMsg += "New entry created: " + blue(p.EntryKey) + "\n"
	} else {
		userMsg += "Entry updated: " + blue(p.EntryKey) +  "\n"
	}

	userMsg += "Configuration imported: " + toPrintKey

	_, _ = fmt.Fprintln(o, userMsg)
	return nil, 0
}

