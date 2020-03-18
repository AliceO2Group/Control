/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: George Raduta <george.raduta@cern.ch>
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
	"encoding/json"
	"errors"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
)

var(
	blue = color.New(color.FgHiBlue).SprintFunc()
	red = color.New(color.FgHiRed).SprintFunc()
	inputComponentEntryRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+)(\/[a-z-A-Z0-9-]+)$`)
)


func isInputCompEntryValid(input string) bool {
	return inputComponentEntryRegex.MatchString(input)
}

func getComponentEntryFromUserInput(input string) (string, string, error) {
	if isInputCompEntryValid(input) {
		splitCom := strings.Split(input, "/")
		return splitCom[0], splitCom[1], nil
	} else {
		return "", "", errors.New(invalidArgsErrMsg)
	}
}

// Method to return a list of components, entries or entries with latest timestamp
// If no keys were passed an error and code exit 3 will be returned
func getListOfComponentsAndOrWithTimestamps(keys []string, keyPrefix string, useTimestamp bool)([]string, error, int) {
	if len(keys) == 0 {
		return []string{},  errors.New("no keys found"), emptyData
	}

	var components []string
	componentsSet := make(map[string]string)

	for _, key := range keys {
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, "/")
		componentTimestamp := componentParts[len(componentParts) - 1]

		if len(componentParts) == 1 {
			componentTimestamp = "unversioned"
		}
		if useTimestamp{
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
	return components, nil, nonZero
}

func drawTableHistoryConfigs(headers []string, history []string, max int, o io.Writer) {
	table := tablewriter.NewWriter(o)
	if len(headers) > 0 {
		table.SetHeader(headers)
	}
	table.SetBorder(false)
	table.SetColMinWidth(0, max)

	for _, value := range history {
		component, entry, timestamp := componentcfg.GetComponentEntryTimestampFromConsul(value)
		prettyTimestamp, err := componentcfg.GetTimestampInFormat(timestamp, time.RFC822)
		if err != nil {
			prettyTimestamp = timestamp
		}
		configName := red(component) + "/" + blue(entry) + "@" + timestamp
		table.Append([]string{configName, prettyTimestamp})
	}
	table.Render()
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
		return
	}
	return parsedOutput, nil
}

func getMaxLenOfKey(keys []string) (maxLen int){
	maxLen = 0
	for _, value := range keys {
		if len(value) - len(componentcfg.ConfigComponentsPath) >= maxLen {
			maxLen = len(value) - len(componentcfg.ConfigComponentsPath)
		}
	}
	return
}

func getFileContent(filePath string)(fileContent []byte, err error) {
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

func isFileExtensionValid(extension string) bool{
	extension = strings.ToUpper(extension)
	return extension == "JSON" || extension == "YAML" || extension == "YML" || extension == "INI" || extension == "TOML"
}
