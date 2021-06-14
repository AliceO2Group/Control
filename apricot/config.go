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

package apricot

import (
	"errors"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"golang.org/x/sys/unix"
)

func setDefaults() error {
	viper.Set("component", "apricot")

	viper.SetDefault("listenPort", 32101)
	viper.SetDefault("httpListenPort", 32188)
	viper.SetDefault("backendUri", "consul://127.0.0.1:8500")
	viper.SetDefault("runtimeBasePath", "o2/runtime")
	viper.SetDefault("workingDir", "/var/lib/o2/apricot")
	viper.SetDefault("verbose", false)
	return nil
}

func setFlags() error {
	pflag.Int("listenPort", viper.GetInt("listenPort"), "Port of apricot server")
	pflag.Int("httpListenPort", viper.GetInt("listenPort"), "Port of apricot http server")
	pflag.String("backendUri", viper.GetString("backendUri"), "URI of the Consul server or YAML configuration file")
	pflag.Bool("verbose", viper.GetBool("verbose"), "Verbose logging")

	pflag.Parse()
	return viper.BindPFlags(pflag.CommandLine)
}

func checkWorkingDirRights() error {
	err := unix.Access(viper.GetString("workingDir"), unix.W_OK)
	if err != nil {
		return errors.New("No write access for core working path \"" + viper.GetString("workingDir") + "\": "+ err.Error())
	}
	return nil
}

// Remove trailing '/'
func sanitizeWorkingPath() {
	sanitizeWorkingPath := filepath.Clean(viper.GetString("workingDir"))
	viper.Set("workingDir", sanitizeWorkingPath)
}

// Bind environment variables with the prefix APRICOT
// e.g. APRICOT_LISTENPORT
func bindEnvironmentVariables() {
	viper.SetEnvPrefix("APRICOT")
	viper.AutomaticEnv()
}


// NewConfig is the constructor for a new config.
func NewConfig() (err error) {
	if err = setDefaults(); err != nil {
		return
	}
	if err = setFlags(); err != nil {
		return
	}
	bindEnvironmentVariables()
	sanitizeWorkingPath()
	if err = checkWorkingDirRights(); err != nil {
		return
	}

	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Trigger apricot backend setup
	// this must happen after viper is ready
	Instance()
	return
}
