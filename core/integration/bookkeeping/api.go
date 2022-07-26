/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
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

package bookkeeping

import (
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	clientAPI "github.com/AliceO2Group/Bookkeeping/go-api-client/src"
	sw "github.com/AliceO2Group/Bookkeeping/go-api-client/src/go-client-generated"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// [O2-2512]: Until JWT becomes optional or provided by BK
// const jwtToken = "token"

var log = logger.New(logrus.StandardLogger(), "bookkeepingclient")

type BookkeepingWrapper struct{}

var (
	once sync.Once
	// mock instance
	instance *BookkeepingWrapper
)

func Instance() *BookkeepingWrapper {
	once.Do(func() {
		apiUrl, err := url.Parse(viper.GetString("bookkeepingBaseUri"))
		if err == nil {
			apiUrl.Path = path.Join(apiUrl.Path + "api")
			clientAPI.InitializeApi(apiUrl.String(), viper.GetString("bookkeepingToken"))
		} else {
			log.WithField("error", err.Error()).
				Error("unable to parse the Bookkeeping base URL")
			clientAPI.InitializeApi(path.Join(viper.GetString("bookkeepingBaseUri")+"api"), viper.GetString("bookkeepingToken"))
		}
		instance = &BookkeepingWrapper{}
	})
	return instance
}

func (bk *BookkeepingWrapper) CreateRun(activityId string, nDetectors int, nEpns int, nFlps int, runNumber int32, runType string, ddFlp bool, dcs bool, epn bool, epnTopology string, detectors string) error {
	var runtypeAPI sw.RunType
	switch runType {
	case string(sw.TECHNICAL_RunType):
		runtypeAPI = sw.TECHNICAL_RunType
	case string(sw.COSMICS_RunType):
		runtypeAPI = sw.COSMICS_RunType
	case string(sw.PHYSICS_RunType):
		runtypeAPI = sw.PHYSICS_RunType
	default:
		// log Runtype is %s and it is not valid overwrite with TECHNICAL_RunType
		runtypeAPI = sw.TECHNICAL_RunType
	}

	_, _, err := clientAPI.CreateRun(activityId, int32(nDetectors), int32(nEpns), int32(nFlps), runNumber, runtypeAPI, ddFlp, dcs, epn, epnTopology, sw.Detectors(detectors))

	return err
}

func (bk *BookkeepingWrapper) UpdateRun(runNumber int32, runResult string, timeO2Start string, timeO2End string, timeTrgStart string, timeTrgEnd string, trgGlobal bool, trg bool, pdpConfig string, pdpTopology string, tfbMode string, odcTopologyFull string, lhcPeriod string) error {
	var runquality sw.RunQuality
	switch runResult {
	case string(sw.GOOD_RunQuality):
		runquality = sw.GOOD_RunQuality
	case string(sw.BAD_RunQuality):
		runquality = sw.BAD_RunQuality
	default:
		runquality = sw.TEST_RunQuality
	}

	timeO2S, ok := strconv.ParseInt(timeO2Start, 10, 64)
	if ok != nil {
		log.WithField("runNumber", runNumber).
			WithField("time", timeO2Start).
			Warning("cannot parse O2 start time")
		timeO2S = -1
	}
	if timeO2Start == "" || timeO2S <= 0 {
		timeO2S = -1
	}
	var timeO2E int64 = -1
	if timeO2End != "" {
		timeO2E, ok = strconv.ParseInt(timeO2End, 10, 64)
		if ok != nil {
			log.WithField("runNumber", runNumber).
				WithField("time", timeO2End).
				Warning("cannot parse O2 end time")
			timeO2E = -1
		}
	}
	if timeO2End == "" || timeO2E <= 0 {
		timeO2E = -1
	}
	var timeTrgS int64 = -1
	var timeTrgE int64 = -1
	if trg {
		timeTrgS, ok = strconv.ParseInt(timeTrgStart, 10, 64)
		if ok != nil {
			log.WithField("runNumber", runNumber).
				WithField("time", timeTrgStart).
				Warning("cannot parse Trg start time")
			timeTrgS = -1
		}
		if timeTrgStart == "" || timeTrgS <= 0 {
			timeTrgS = -1
		}
		timeTrgE, ok = strconv.ParseInt(timeTrgEnd, 10, 64)
		if ok != nil {
			log.WithField("runNumber", runNumber).
				WithField("time", timeTrgEnd).
				Warning("cannot parse Trg end time")
			timeTrgE = -1
		}
		if timeTrgEnd == "" || timeTrgE <= 0 {
			timeTrgE = -1
		}
	}

	_, _, err := clientAPI.UpdateRun(runNumber, runquality, timeO2S, timeO2E, timeTrgS, timeTrgE, trgGlobal, trg, pdpConfig, pdpTopology, tfbMode, odcTopologyFull, lhcPeriod)
	return err
}

func (bk *BookkeepingWrapper) CreateLog(text string, title string, runNumbers string, parentLogId int32) error {
	_, _, err := clientAPI.CreateLog(text, title, runNumbers, parentLogId)
	return err
}

func (bk *BookkeepingWrapper) CreateFlp(name string, hostName string, runNumber int32) error {
	_, _, err := clientAPI.CreateFlp(name, hostName, runNumber)
	return err
}

func (bk *BookkeepingWrapper) UpdateFlp(name string, runNumber int32, nSubtimeframes int32, nEquipmentBytes int32, nRecordingBytes int32, nFairMQBytes int32) error {
	_, _, err := clientAPI.UpdateFlp(name, runNumber, nSubtimeframes, nEquipmentBytes, nRecordingBytes, nFairMQBytes)
	return err
}

func (bk *BookkeepingWrapper) GetLogs() {
	clientAPI.GetLogs()
	log.Debug("GetLogs call done")
}

func (bk *BookkeepingWrapper) GetRuns() {
	clientAPI.GetRuns()
	log.Debug("GetRuns call done")
}

func (bk *BookkeepingWrapper) CreateEnvironment(envId string, createdAt time.Time, status string, statusMessage string) error {
	_, _, err := clientAPI.CreateEnvironment(envId, createdAt, status, statusMessage)
	return err
}

func (bk *BookkeepingWrapper) UpdateEnvironment(envId string, toredownAt time.Time, status string, statusMessage string) error {
	_, _, err := clientAPI.UpdateEnvironment(envId, toredownAt, status, statusMessage)
	return err
}
