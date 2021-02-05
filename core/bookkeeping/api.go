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

// This package instantiates the bookkeeping API client. There is one
// issue with the local import in github.com/AliceO2Group/Bookkeeping/go-api-client/src
// which should be fixed, or otherwise we need to mirror the BookkeepingAPI.go here.
// The other possible issue is that currently all the call instead of returning the
// responses and the errors they use fmt.Println(). This can be solved again by mirroring,
// and changing the implementation of the BookkeepingAPI.go to return instead of printing,
// but this is not a valid solution since we have to reflect every change that might occur
// on the file.
package bookkeeping

import (
	"sync"

	clientAPI "github.com/AliceO2Group/Bookkeeping/go-api-client/src"
	sw "github.com/AliceO2Group/Bookkeeping/go-api-client/src/go-client-generated"
)

const (
	bookkeepingBaseUrl       = "http://vm4.jiskefet.io/api"
	// APItoken provided doesn't work, we should contact team on how to generate one.
	bookkeepingAPIToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6NzQ2NjQwLCJ1c2VybmFtZSI6ImF3ZWdyenluIiwiYWNjZXNzIjoxLCJpYXQiOjE2MTA1NDM0ODYsImV4cCI6MTYxMDYyOTg4NiwiaXNzIjoibzItdWkifQ.8pM1K0HIfpnZop7bJk_rD5GvkfeiWaNNs2S7ZM1tqYg"
)

type BookkeepingWrapper struct {}

var (
	once sync.Once
	// mock instance
	instance *BookkeepingWrapper
)
func Instance() *BookkeepingWrapper {
	once.Do(func() {
		clientAPI.InitializeApi(bookkeepingBaseUrl, bookkeepingAPIToken)
	})
	return instance
}

func initializeBookkeepingAPI() *BookkeepingWrapper {
	var bkw BookkeepingWrapper
	clientAPI.InitializeApi(bookkeepingBaseUrl, bookkeepingAPIToken)
	return &bkw

}


func (bk *BookkeepingWrapper) CreateRun(activityId string, nDetectors int, nEpns int, nFlps int, runNumber int, runType string, timeO2Start int64, timeTrgStart int64){
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

	clientAPI.CreateRun(activityId, int64(nDetectors),int64(nEpns),int64(nFlps),int64(runNumber), runtypeAPI, timeO2Start, timeTrgStart)

}

func (bk *BookkeepingWrapper) UpdateRun(runNumber int, runResult string, timeO2End int64, timeTrgEnd int64){
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

	clientAPI.UpdateRun(int64(runNumber), runquality, timeO2End, timeTrgEnd)
}


func (bk *BookkeepingWrapper) CreateLog(text string, title string, runNumbers string, parentLogId int64){
	clientAPI.CreateLog(text, title, runNumbers, parentLogId)
}


func (bk *BookkeepingWrapper) GetLogs(){
	clientAPI.GetLogs()
}

func (bk *BookkeepingWrapper) GetRuns(){
	clientAPI.GetRuns()
}