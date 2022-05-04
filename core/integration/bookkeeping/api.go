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

func (bk *BookkeepingWrapper) CreateRun(activityId string, nDetectors int, nEpns int, nFlps int, runNumber int32, runType string, timeO2Start time.Time, timeTrgStart time.Time, dd_flp bool, dcs bool, epn bool, epnTopology string, detectors string) {
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

	arrayResponse, response, err := clientAPI.CreateRun(activityId, int32(nDetectors), int32(nEpns), int32(nFlps), runNumber, runtypeAPI, timeO2Start, timeTrgStart, dd_flp, dcs, epn, epnTopology, sw.Detectors(detectors))
	if err != nil || arrayResponse.Data.RunNumber != runNumber {
		log.WithError(err).
			WithField("runType", runType).
			WithField("partition", activityId).
			WithField("runNumber", runNumber).
			WithField("response", arrayResponse.Data.RunNumber).
			WithField("httpresponse", response.Status).
			WithField("call", "CreateRun").
			Error("Bookkeeping API CreateRun error")
	} else {
		log.WithField("runType", runType).
			WithField("partition", activityId).
			WithField("runNumber", runNumber).
			WithField("httpresponse", response.Status).
			Debug("CreateRun call successful")
	}
}

func (bk *BookkeepingWrapper) UpdateRun(runNumber int32, runResult string, timeO2End time.Time, timeTrgEnd time.Time) {
	var runquality sw.RunQuality
	switch runResult {
	case string(sw.GOOD_RunQuality):
		runquality = sw.GOOD_RunQuality
	case string(sw.BAD_RunQuality):
		runquality = sw.BAD_RunQuality
	//case string(sw.UNKNOWN_RunQuality):
	//	runquality = sw.UNKNOWN_RunQuality
	default:
		// log runquality is %s and it is not valid.  overwrite with UNKNOWN_RunQuality
		runquality = sw.TEST_RunQuality
	}

	arrayResponse, response, err := clientAPI.UpdateRun(runNumber, runquality, timeO2End, timeTrgEnd)
	if err != nil || arrayResponse.Data.RunNumber != runNumber {
		log.WithError(err).
			WithField("runNumber", runNumber).
			WithField("response", arrayResponse.Data.RunNumber).
			WithField("httpresponse", response.Status).
			WithField("call", "UpdateRun").
			Error("Bookkeeping API UpdateRun error")
	} else {
		log.WithField("runNumber", runNumber).
			WithField("httpresponse", response.Status).
			Debug("UpdateRun call successful")
	}
}

func (bk *BookkeepingWrapper) CreateLog(text string, title string, runNumbers string, parentLogId int32) {
	arrayResponse, response, err := clientAPI.CreateLog(text, title, runNumbers, parentLogId)
	if err != nil || arrayResponse.Data.Title != title {
		log.WithError(err).
			WithField("title", title).
			WithField("response", arrayResponse.Data.Title).
			WithField("httpresponse", response.Status).
			WithField("call", "CreateLog").
			Error("Bookkeeping API CreateLog error")
	} else {
		log.WithField("title", title).
			WithField("httpresponse", response.Status).
			Debug("CreateLog call successful")
	}
}

func (bk *BookkeepingWrapper) CreateFlp(name string, hostName string, runNumber int32) {
	arrayResponse, response, err := clientAPI.CreateFlp(name, hostName, runNumber)
	if err != nil {
		log.WithError(err).
			WithField("runNumber", runNumber).
			WithField("response", arrayResponse.Data.Title).
			WithField("httpresponse", response.Status).
			WithField("call", "CreateFlp").
			Error("Bookkeeping API CreateFlp error")
	} else {
		log.WithField("runNumber", runNumber).
			WithField("httpresponse", response.Status).
			Debug("CreateFlp call successful")
	}
}

func (bk *BookkeepingWrapper) UpdateFlp(name string, runNumber int32, nSubtimeframes int32, nEquipmentBytes int32, nRecordingBytes int32, nFairMQBytes int32) {
	arrayResponse, response, err := clientAPI.UpdateFlp(name, runNumber, nSubtimeframes, nEquipmentBytes, nRecordingBytes, nFairMQBytes)
	if err != nil || arrayResponse.Data.Name != name {
		log.WithError(err).
			WithField("runNumber", runNumber).
			WithField("name", name).
			WithField("response", arrayResponse.Data.Name).
			WithField("httpresponse", response.Status).
			WithField("call", "UpdateFlp").
			Error("Bookkeeping API UpdateFlp error")
	} else {
		log.WithField("runNumber", runNumber).
			WithField("httpresponse", response.Status).
			Debug("UpdateFlp call successful")
	}
}

func (bk *BookkeepingWrapper) GetLogs() {
	clientAPI.GetLogs()
	log.Debug("GetLogs call done")
}

func (bk *BookkeepingWrapper) GetRuns() {
	clientAPI.GetRuns()
	log.Debug("GetRuns call done")
}

func (bk *BookkeepingWrapper) CreateEnvironment(envId string, createdAt time.Time, status string, statusMessage string) {
	arrayResponse, response, err := clientAPI.CreateEnvironment(envId, createdAt, status, statusMessage)
	if err != nil || arrayResponse.Data.Id != envId {
		log.WithError(err).
			WithField("environment", envId).
			WithField("response", arrayResponse.Data.Id).
			WithField("httpresponse", response.Status).
			WithField("call", "CreateEnvironment").
			Error("Bookkeeping API CreateEnvironment error")
	} else {
		log.WithField("environment", envId).
			WithField("httpresponse", response.Status).
			Debug("CreateEnvironment call successful")
	}
}

func (bk *BookkeepingWrapper) UpdateEnvironment(envId string, toredownAt time.Time, status string, statusMessage string) {
	arrayResponse, response, err := clientAPI.UpdateEnvironment(envId, toredownAt, status, statusMessage)
	if err != nil || arrayResponse.Data.Id != envId {
		log.WithError(err).
			WithField("environment", envId).
			WithField("response", arrayResponse.Data.Id).
			WithField("httpresponse", response.Status).
			WithField("call", "UpdateEnvironment").
			Error("Bookkeeping API UpdateEnvironment error")
	} else {
		log.WithField("environment", envId).
			WithField("httpresponse", response.Status).
			Debug("UpdateEnvironment call successful")
	}
}
