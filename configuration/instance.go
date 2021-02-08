/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package configuration

import (
	"sync"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	instance *Service
)

func Instance() *Service {
	once.Do(func() {
		var(
			err error
			configUri string
		)
		if viper.IsSet("config_endpoint") { //coconut
			configUri = viper.GetString("config_endpoint")
		} else {
			configUri = viper.GetString("globalConfigurationUri")
		}
		instance, err = newService(configUri)
		if err != nil {
			log.WithField("globalConfigurationUri", configUri).Fatal("bad configuration URI")
		}
	})
	return instance
}

