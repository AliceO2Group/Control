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

var log = logger.New(logrus.StandardLogger(), "Bookkeeping")

type BookkeepingWrapper struct{}

var (
	once sync.Once
	// mock instance
	instance *BookkeepingWrapper
)

func Instance() *BookkeepingWrapper {
	once.Do(func() {
		apiUrl, err  := url.Parse(viper.GetString("bookkeepingBaseUri"))
		if err == nil {
			apiUrl.Path = path.Join( apiUrl.Path + "api")
			clientAPI.InitializeApi(apiUrl.String(), getJWTAPIToken())
		} else {
			log.WithField("error", err.Error()).
			Error("unable to parse the Bookkeeping base URL")
			clientAPI.InitializeApi(viper.GetString("bookkeepingBaseUri")+"/api", getJWTAPIToken())
		}
	})
	return instance
}

func (bk *BookkeepingWrapper) CreateRun(activityId string, nDetectors int, nEpns int, nFlps int, runNumber int32, runType string, timeO2Start time.Time, timeTrgStart time.Time) {
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

	clientAPI.CreateRun(activityId, int32(nDetectors), int32(nEpns), int32(nFlps), runNumber, runtypeAPI, timeO2Start, timeTrgStart)
}

func (bk *BookkeepingWrapper) UpdateRun(runNumber int32, runResult string, timeO2End time.Time, timeTrgEnd time.Time) {
	var runquality sw.RunQuality
	switch runResult {
	case string(sw.GOOD_RunQuality):
		runquality = sw.GOOD_RunQuality
	case string(sw.BAD_RunQuality):
		runquality = sw.BAD_RunQuality
	case string(sw.UNKNOWN_RunQuality):
		runquality = sw.UNKNOWN_RunQuality
	default:
		// log runquality is %s and it is not valid.  overwrite with UNKNOWN_RunQuality
		runquality = sw.UNKNOWN_RunQuality
	}

	clientAPI.UpdateRun(runNumber, runquality, timeO2End, timeTrgEnd)
}

func (bk *BookkeepingWrapper) CreateLog(text string, title string, runNumbers string, parentLogId int32) {
	clientAPI.CreateLog(text, title, runNumbers, parentLogId)
}

func (bk *BookkeepingWrapper) CreateFlp(name string, hostName string, runNumber int32) {
	clientAPI.CreateFlp(name, hostName, runNumber)
}

func (bk *BookkeepingWrapper) UpdateFlp(flpId int32, name string, nSubtimeframes int32, nEquipmentBytes int32, nRecordingBytes int32, nFairMQBytes int32) {
	clientAPI.UpdateFlp(flpId, name, nSubtimeframes, nEquipmentBytes, nRecordingBytes, nFairMQBytes)
}

func (bk *BookkeepingWrapper) GetLogs() {
	clientAPI.GetLogs()
}

func (bk *BookkeepingWrapper) GetRuns() {
	clientAPI.GetRuns()
}
