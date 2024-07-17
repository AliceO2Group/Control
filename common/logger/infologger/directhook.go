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

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/common/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const INFOLOGGER_MAX_MESSAGE_SIZE = 1024

var (
	hostname string
	Pid      string
	username string

	logILInfoLevel = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
	logILAllLevel  = logrus.AllLevels
	currentIlLevel = logILInfoLevel
)

func setCurrentILLevelFromViper() {
	if viper.GetBool("logAllIL") {
		currentIlLevel = logILAllLevel
	} else {
		currentIlLevel = logILInfoLevel
	}
}

var lineBreaksRe = regexp.MustCompile(`\r?\n`)

func init() {
	var err error
	Pid = fmt.Sprintf("%d", os.Getpid())

	unixUser, _ := user.Current()
	if unixUser != nil {
		username = unixUser.Username
	}

	hostname, err = os.Hostname()
	if err != nil {
		return
	}

	// We only take the short hostname
	hostname = strings.Split(hostname, ".")[0]
}

type sender struct {
	stream net.Conn
}

func newSender(path string) *sender {
	stream, err := net.Dial("unix", path)
	if err != nil {
		fmt.Printf("cannot dial unix socket %s\n", path)
		return nil
	}
	return &sender{
		stream: stream,
	}
}

func (s *sender) Close() error {
	return s.stream.Close()
}

func (s *sender) format(fields map[string]string, version protoVersion) string {
	stringLog := string("*" + version)
	currentProtocol := protocols[version]
	for _, fSpec := range *currentProtocol {
		stringLog += "#"
		if value, ok := fields[fSpec.name]; ok {
			stringLog += value
		}
	}

	// We sanitize away all linebreaks from the payload, and then we append one.
	// This is necessary because IL treats \n as a message terminator character.
	stringLog = lineBreaksRe.ReplaceAllString(stringLog, " ")

	return stringLog + "\n"
}

func (s *sender) Send(fields map[string]string) error {
	if s.stream != nil {
		_, err := s.stream.Write([]byte(s.format(fields, v14)))
		return err
	}
	return errors.New("could not send log message: InfoLogger socket not available")
}

type DirectHook struct {
	il       *sender
	system   string
	facility string
	role     string
}

func paddedAbstractSocket(name string) string {
	const targetLen = 108 // Linux constant
	out := make([]byte, targetLen)
	for i := 0; i < targetLen; i++ {
		if i < len(name) {
			out[i] = name[i]
		} else {
			out[i] = '\x00'
		}
	}
	return string(out)
}

func guessSocketPath() string {
	if runtime.GOOS == "linux" {
		return paddedAbstractSocket("@infoLoggerD")
	} else { // macOS
		return "/tmp/infoLoggerD.socket"
	}
}

func NewDirectHook(defaultSystem string, defaultFacility string, levelsToLog []logrus.Level) (*DirectHook, error) {

	if levelsToLog == nil {
		setCurrentILLevelFromViper()
	} else {
		currentIlLevel = levelsToLog
	}

	socketPath := guessSocketPath()
	sender := newSender(socketPath)
	if sender == nil {
		return nil, fmt.Errorf("cannot instantiate InfoLogger hook on socket %s", socketPath)
	}
	return &DirectHook{
		il:       sender,
		system:   defaultSystem,
		facility: defaultFacility,
		role:     hostname,
	}, nil
}

func NewDirectHookWithRole(defaultSystem string, defaultFacility string, defaultRole string, levelsToLog []logrus.Level) (*DirectHook, error) {
	dh, err := NewDirectHook(defaultSystem, defaultFacility, levelsToLog)
	if dh != nil {
		dh.role = defaultRole
	}
	return dh, err
}

func (h *DirectHook) Levels() []logrus.Level {
	return currentIlLevel
}

func (h *DirectHook) Fire(e *logrus.Entry) error {
	// Supported field names:
	// 		Severity, Level, ErrorCode, SourceFile, SourceLine,
	// 		Facility, Role, System, Detector, Partition, Run
	// Filled automatically by InfoLogger, do not set: PID, hostName, userName
	payload := make(map[string]string)
	payload["severity"] = logrusLevelToInfoLoggerSeverity(e.Level)
	if _, hasLevel := payload["level"]; !hasLevel {
		payload["level"] = logrusEntryToInfoLoggerLevel(e)
	}
	payload["timestamp"] = utils.NewUnixTimestamp()
	payload["hostname"] = hostname
	payload["pid"] = Pid
	payload["username"] = username

	if e.HasCaller() {
		payload["errsource"] = e.Caller.File
		payload["errline"] = strconv.Itoa(e.Caller.Line)
	}

	var message strings.Builder
	message.WriteString(e.Message)

	unmappableFields := make(sort.StringSlice, 0)

	if e.Data != nil {
		for k, v := range e.Data {
			if k == "prefix" {
				continue
			}

			if k == "nohooks" {
				if vBool, ok := v.(bool); ok {
					if vBool {
						return nil
					}
				}
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
					payload[Facility] = buildFineFacility(vStr, e.Data)
				} else {
					payload[k] = vStr
				}
			} else { // data structure key not mappable to InfoLogger field
				unmappableFields = append(unmappableFields, fmt.Sprintf(" %s=\"%s\"", k, vStr))
			}
		}
	}

	// If a System, Facility or Role isn't passed manually, we fall back on defaults
	if _, ok := e.Data[System]; !ok {
		payload[System] = h.system
	}
	if _, ok := e.Data[Facility]; !ok {
		payload[Facility] = buildFineFacility(h.facility, e.Data)
	}
	if _, ok := e.Data[Role]; !ok {
		payload[Role] = h.role
	}

	unmappableFields.Sort()

	for _, v := range unmappableFields {
		message.WriteString(v)
	}

	messageStr := message.String()

	// IL won't be happy if we send a message longer than 1024 bytes
	if len(messageStr) > INFOLOGGER_MAX_MESSAGE_SIZE {
		messageStr = messageStr[:INFOLOGGER_MAX_MESSAGE_SIZE]
	}

	payload["message"] = messageStr

	if h.il != nil {
		err := h.il.Send(payload)
		if err != nil {
			return fmt.Errorf("infoLogger hook error: %s", err.Error())
		}
	} else {
		return fmt.Errorf("infoLogger hook error: sender not available")
	}
	return nil
}
