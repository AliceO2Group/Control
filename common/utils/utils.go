/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2019 CERN and copyright holders of ALICE O².
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

package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TimeTrack(start time.Time, name string, log *logrus.Entry) {
	if !viper.GetBool("verbose") {
		return
	}

	if log == nil {
		log = logger.New(logrus.StandardLogger(), "debug").WithPrefix("debug")
	}
	elapsed := time.Since(start)
	log.WithField("level", 11 /*devel*/).Debugf("%s took %s", name, elapsed)
}

func TimeTrackFunction(start time.Time, log *logrus.Entry) {
	// Skip this function, and fetch the PC and file for its parent.
	pc, _, _, _ := runtime.Caller(1)

	// Retrieve a function object this functions parent.
	funcObj := runtime.FuncForPC(pc)

	// Regex to extract just the function name (and not the module path).
	runtimeFunc := regexp.MustCompile(`^.*\.(.*)$`)
	name := runtimeFunc.ReplaceAllString(funcObj.Name(), "$1")
	log = log.WithField("method", funcObj.Name())

	TimeTrack(start, name, log)
}

func NewUnixTimestamp() string {
	// User for IL direct hook and scheduler.go
	return fmt.Sprintf("%f", float64(time.Now().UnixNano())/1e9)
}

func IsDirEmpty(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close() //Read-only we don't care about the return value

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func EnsureTrailingSlash(path *string) {
	if !strings.HasSuffix(*path, "/") { //Add trailing '/'
		*path += "/"
	}
}

// helper func to package strings up nicely for protobuf
func ProtoString(s string) *string { return &s }

func StringSliceContains(s []string, str string) bool {
	for _, a := range s {
		if a == str {
			return true
		}
	}
	return false
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

func ParseExtraVars(extraVars string) (extraVarsMap map[string]string, err error) {
	extraVarsMap = make(map[string]string)
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
	return
}

func TruncateString(str string, length int) string {
	if length <= 0 {
		return ""
	}

	if utf8.RuneCountInString(str) < length {
		return str
	}

	return string([]rune(str)[:length])
}

var taskClassNameRgx = regexp.MustCompile(`(?:.*/)?([^/@]+)@`) // Captures the segment between the last '/' and '@'

// ExtractTaskClassName extracts the task class name from the provided task name string
// For example, we extract "readout" from the following string:
// "alio2-cr1-hv-gw01.cern.ch:/opt/git/ControlWorkflows/tasks/readout@12b11ac4bb652e1835e3e94806a688c951691d5f#2sP21PjpfCQ"
func ExtractTaskClassName(taskName string) (string, error) {

	matches := taskClassNameRgx.FindStringSubmatch(taskName)
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to extract task class name from '%s'", taskName)
	}

	return matches[1], nil
}

// TrimJitPrefix removes the JIT prefix from task class names.
// For example, "jit-ad6f2b64b7502198430d7d7f93f15bf94c088cab-qc-pp-TPC-CalibQC_long" becomes "qc-pp-TPC-CalibQC_long".
// If input does not contain a JIT prefix, it is returned as it is.
func TrimJitPrefix(taskClassName string) string {
	if strings.HasPrefix(taskClassName, "jit-") {
		parts := strings.SplitN(taskClassName, "-", 3)
		if len(parts) > 2 {
			return parts[2]
		}
	}
	return taskClassName
}
