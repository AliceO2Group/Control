/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"strconv"
	"time"
	"io"
	"io/ioutil"
	"os"
	"fmt"
	"strings"
	"errors"
	"regexp"
	"encoding/json"
	"gopkg.in/yaml.v2"
)

var  (
	inputFullRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+)(\/[a-z-A-Z0-9-]+){1}(\@[0-9]+)?$`)
	inputCompEntryRegex = regexp.MustCompile(`^([a-zA-Z0-9-]+)(\/[a-z-A-Z0-9-]+){1}$`)
)
var(
	blue = color.New(color.FgHiBlue).SprintFunc()
	red = color.New(color.FgHiRed).SprintFunc()
)

func IsInputCompEntryTsValid(input string) bool {
	return inputFullRegex.MatchString(input)
}

func isInputComponentEntryValid(input string) bool {
	return inputCompEntryRegex.MatchString(input)
}

func IsInputSingleValidWord(input string) bool {
	return !strings.Contains(input, "/") && !strings.Contains(input, "@")
}

// Method to parse a timestamp in the specified format
func GetTimestampInFormat(timestamp string, timeFormat string)(string, error){
	timeStampAsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Unable to identify timestamp"))
	}
	tm := time.Unix(timeStampAsInt, 0)
	return  tm.Format(timeFormat), nil
}

// Method to return the latest timestamp for a specified component & entry
// If no keys were passed an error and code exit 3 will be returned
func GetLatestTimestamp(keys []string, component string, entry string)(timestamp string, err error, code int) {
	keyPrefix := componentsPath + component + "/" + entry
	if len(keys) == 0 {
		err = errors.New(fmt.Sprintf("No keys found"))
		return "", err, emptyData
	}

	var maxTimeStamp uint64
	for _, key := range keys {
		componentTimestamp, err := strconv.ParseUint(strings.TrimPrefix(key, keyPrefix + "/"), 10, 64)
		if err == nil {
			if componentTimestamp > maxTimeStamp  {
				maxTimeStamp = componentTimestamp
			}
		}
	}
	return strconv.FormatUint(maxTimeStamp, 10), nil, nonZero
}

// Method to return a list of components, entries or entries with latest timestamp
// If no keys were passed an error and code exit 3 will be returned
func GetListOfComponentsAndOrWithTimestamps(keys []string, keyPrefix string, useTimestamp bool)([]string, error, int) {
	if len(keys) == 0 {
		return []string{},  errors.New(fmt.Sprintf("No keys found")), emptyData
	}

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
		component, entry, timestamp := GetComponentEntryTimestampFromConsul(value)
		prettyTimestamp, err := GetTimestampInFormat(timestamp, time.RFC822)
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

// Method to split component, entry and timestamp when being passed a key from consul
// e.g. of key o2/components/quality-control/cru-demo/12345678
func GetComponentEntryTimestampFromConsul(key string)(string, string, string) {
	key = strings.TrimPrefix(key, componentsPath)
	key = strings.TrimPrefix(key, "/'")
	key = strings.TrimSuffix(key, "/")
	elements := strings.Split(key, "/")
	return elements[0], elements[1], elements[2]
}

func GetMaxLenOfKey(keys []string) (maxLen int){
	maxLen = 0
	for _, value := range keys {
		if len(value) - len(componentsPath) >= maxLen {
			maxLen = len(value) - len(componentsPath)
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

func getComponentsMapFromKeysList(keys []string) map[string]bool {
	var componentsMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, componentsPath)
		component := strings.Split(value, "/" )[0]
		componentsMap[component] = true
	}
	return componentsMap
}

func getEntriesMapOfComponentFromKeysList(component string, keys []string) map[string]bool  {
	var entriesMap = make(map[string]bool)
	for _,value := range keys {
		value := strings.TrimPrefix(value, componentsPath)
		parts := strings.Split(value, "/" )
		if parts[0] == component {
			entriesMap[parts[1]] = true
		}
	}
	return entriesMap
}

func isFileExtensionValid(extension string) bool{
	extension = strings.ToUpper(extension)
	return extension == "JSON" || extension == "YAML" || extension == "INI" || extension == "TOML"
}
