/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2017-2018 CERN and copyright holders of ALICE O².
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

package main

import (
	"fmt"
	"os"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/executor"
	"github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetOutput(os.Stdout)
	ilHook, err := infologger.NewDirectHook("ECS", "executor", logrus.AllLevels)
	if err == nil {
		logrus.AddHook(ilHook)
	}
}

var log = logger.New(logrus.StandardLogger(), "executor")

// Entry point, reads configuration from environment variables.
func main() {
	logrus.SetLevel(logrus.DebugLevel)
	infologger.Pid = fmt.Sprintf("%d", os.Getpid())

	cfg, err := config.FromEnv()
	if err != nil {
		log.WithField("error", err.Error()).Fatal("failed to load configuration")
	}
	log.WithField("configuration", cfg).Debug("configuration loaded")
	executor.Run(cfg)
	log.WithField("executorId", cfg.ExecutorID).Info("executor exiting")
	os.Exit(0)
}
