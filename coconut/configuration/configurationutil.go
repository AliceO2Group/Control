/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: George Raduta <george.raduta@cern.ch>
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

package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var (
	blue = color.New(color.FgHiBlue).SprintFunc()
	red  = color.New(color.FgHiRed).SprintFunc()
	//                                                 component        /RUNTYPE          /rolename             /entry
	inputComponentEntryRegex = regexp.MustCompile(`^([a-zA-Z0-9-_]+)(\/[A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}$`)
)

// Utility function to create a componentcfg.Query from combos of args and flags
func queryFromFlags(cmd *cobra.Command, args []string) (query *componentcfg.Query, err error) {
	if len(args) < 1 || len(args) > 2 {
		err = errors.New(fmt.Sprintf("accepts 1 or 2 arg(s), but received %d", len(args)))
		return
	}

	switch len(args) {
	case 1: // coconut conf show component/RUNTYPE/role/entry
		if componentcfg.IsStringValidQueryPath(args[0]) {
			if strings.Contains(args[0], "/") {
				// coconut conf show c/R/r/e
				query, err = componentcfg.NewQuery(args[0])
				if err != nil {
					return
				}
			}
		} else {
			err = errors.New("please provide entry name")
			return
		}
	case 2: // coconut conf show component entry [--runtype RUNTYPE_EXPR] [--role role_expr] [--timestamp]
		var runTypeS, machineRole string
		var runType apricotpb.RunType
		runTypeS, err = cmd.Flags().GetString("runtype")
		if err != nil {
			return
		}
		if len(runTypeS) == 0 {
			runType = apricotpb.RunType_ANY // default value for empty runType input
		} else {
			runTypeI, ok := apricotpb.RunType_value[runTypeS]
			if !ok {
				err = fmt.Errorf("bad value for run type: %s", runTypeS)
				return
			}
			runType = apricotpb.RunType(runTypeI)
		}
		machineRole, err = cmd.Flags().GetString("role")
		if err != nil {
			return
		}
		if len(machineRole) == 0 {
			machineRole = "any" // default value for empty machine role input
		}

		if !componentcfg.IsInputSingleValidWord(args[0]) || !componentcfg.IsInputSingleValidWord(args[1]) {
			err = errors.New(EC_INVALID_ARGS_MSG)
			return
		} else {
			query = &componentcfg.Query{
				Component: args[0],
				RunType:   runType,
				RoleName:  machineRole,
				EntryKey:  args[1],
			}
		}
	}
	return
}

func formatListOutput(cmd *cobra.Command, output []string) (parsedOutput []byte, err error) {
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
		return
	}
	parsedOutput = bytes.TrimSuffix(parsedOutput, []byte("\n"))
	return parsedOutput, nil
}

func getMaxLenOfKey(keys []string) (maxLen int) {
	maxLen = 0
	for _, value := range keys {
		if len(value)-len(componentcfg.ConfigComponentsPath) >= maxLen {
			maxLen = len(value) - len(componentcfg.ConfigComponentsPath)
		}
	}
	return
}

func getFileContent(filePath string) (fileContent []byte, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	fileContentByte, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	return fileContentByte, nil
}

func isFileExtensionValid(extension string) bool {
	extension = strings.ToUpper(extension)
	return extension == "JSON" || extension == "YAML" || extension == "YML" || extension == "INI" || extension == "TOML"
}
