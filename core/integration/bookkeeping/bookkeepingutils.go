/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Miltiadis Alexis <miltiadis.alexis@cern.ch>
 *         Claire Guyot <claire.guyot@cern.ch>
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
	"net/http"

	"github.com/spf13/viper"
)

func getJWTAPIToken() (jwtToken string) {
	req, err := http.NewRequest("GET", viper.GetString("bookkeepingBaseUri"), nil)
	if err != nil {
		log.WithField("error", err.Error()).
			Error("cannot create http GET request")
		return
	}
	client := new(http.Client)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if token, ok := req.URL.Query()["token"]; ok {
			log.WithField("JWT token", token[0]).
				Debug("bookkeeping jwt token")
			jwtToken = token[0]
		}
		return nil
	}

	_, err = client.Do(req)
	if err != nil {
		log.WithField("error", err.Error()).
			Error("cannot execute http request")
		return
	}
	return
}
