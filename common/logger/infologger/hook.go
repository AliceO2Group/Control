/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package infologger

/*
import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"infoLoggerForGo"
)

type Hook struct {
	infoLogger infoLoggerForGo.InfoLogger
	system string
	facility string
}

func NewHook(defaultSystem string, defaultFacility string) (*Hook, error) {
	return &Hook{
		infoLogger: infoLoggerForGo.NewInfoLogger(),
		system: defaultSystem,
		facility: defaultFacility,
	}, nil
}

func (h *Hook) Levels() []logrus.Level {
	// Everything except logrus.TraceLevel
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (h *Hook) Fire(e *logrus.Entry) error {
	// Supported field names:
	// 		Severity, Level, ErrorCode, SourceFile, SourceLine,
	// 		Facility, Role, System, Detector, Partition, Run
	// Filled automatically by InfoLogger, do not set: PID, hostName, userName

	ilMetadata := infoLoggerForGo.NewInfoLoggerMetadata()
	ilMetadata.SetField("Severity", logrusLevelToInfoLoggerSeverity(e.Level))

	if e.HasCaller() {
		ilMetadata.SetField("SourceFile", e.Caller.File)
		ilMetadata.SetField("SourceLine", strconv.Itoa(e.Caller.Line))
	}

	var message strings.Builder
	message.WriteString(e.Message)

	unmappableFields := make(sort.StringSlice, 0)

	if e.Data != nil {
		for k, v := range e.Data {
			if k == "prefix" {
				continue
			}

			var vStr string
			switch v.(type) {
			case string:
				vStr = v.(string)
			case []byte:
				vStr = string(v.([]byte)[:])
			case fmt.Stringer:
				vStr = v.(fmt.Stringer).String()
			default:
				vStr = fmt.Sprintf("%v", v)
			}

			if _, ok := Fields[k]; ok {
				if k == Facility {
					ilMetadata.SetField(k, buildFineFacility(vStr, e.Data))

				} else {
					ilMetadata.SetField(k, vStr)
				}
			} else { // data structure key not mappable to InfoLogger field
				unmappableFields = append(unmappableFields, fmt.Sprintf(" %s=\"%s\"", k, vStr))
			}
		}
	}

	// If a System and/or Facility isn't passed manually, we fall back on defaults
	if _, ok := e.Data[System]; !ok {
		ilMetadata.SetField(System, h.system)
	}
	if _, ok := e.Data[Facility]; !ok {
		ilMetadata.SetField(Facility, buildFineFacility(h.facility, e.Data))
	}

	unmappableFields.Sort()

	for _, v := range unmappableFields {
		message.WriteString(v)
	}

	ilReturn := h.infoLogger.LogM(ilMetadata, message.String())
	if ilReturn != 0 {
		return fmt.Errorf("infoLogger error %d", ilReturn)
	}
	return nil
}
*/
