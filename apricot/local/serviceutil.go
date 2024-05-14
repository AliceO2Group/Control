/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019-2020 CERN and copyright holders of ALICE O².
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

package local

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/AliceO2Group/Control/configuration/cfgbackend"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
)

func (s *Service) queryToAbsPath(query *componentcfg.Query) (absolutePath string, err error) {
	if query == nil {
		return
	}

	absolutePath = query.AbsoluteRaw()
	if exists, _ := s.src.Exists(absolutePath); exists {
		err = nil
		return
	}

	err = fmt.Errorf("no payload at configuration path %s", absolutePath)

	return
}

func (s *Service) getStringMap(path string) map[string]string {
	tree, err := s.src.GetRecursive(path)
	if err != nil {
		log.WithError(err).
			WithField("path", path).
			Warning("getStringMap from configuration backend failed, possibly rate limited")
		return map[string]string{}
	}
	if tree.Type() == cfgbackend.IT_Map {
		responseMap := tree.Map()
		theMap := make(map[string]string, len(responseMap))
		trimSpaces := viper.GetBool("trimSpaceInVarsFromConsulKV")
		for k, v := range responseMap {
			if v.Type() != cfgbackend.IT_Value {
				continue
			}
			if trimmedValue := strings.TrimSpace(v.Value()); v.Value() != trimmedValue {
				valueToStore := v.Value()
				if trimSpaces {
					log.WithError(errors.New("leading or trailing space in value")).
						WithField("path", path).
						WithField("key", k).
						Warning("found unsanitized string value in configuration backend, returning whitespace-trimmed string")
					valueToStore = trimmedValue
				} else {
					log.WithError(errors.New("leading or trailing space in value")).
						WithField("path", path).
						WithField("key", k).
						Warning("found unsanitized string value in configuration backend, returning anyway")
				}
				theMap[k] = valueToStore
			} else {
				theMap[k] = v.Value()
			}
		}
		return theMap
	}
	return map[string]string{}
}

func (s *Service) resolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error) {
	resolved = &componentcfg.Query{}
	*resolved = *query
	if _, err = s.queryToAbsPath(resolved); err == nil {
		// requested path exists, return it
		return
	}

	resolved = query.WithFallbackRunType()
	if _, err = s.queryToAbsPath(resolved); err == nil {
		// path with run type ANY exists, return it
		return
	}

	resolved = query.WithFallbackRoleName()
	if _, err = s.queryToAbsPath(resolved); err == nil {
		// path with role name "any" exists, return it
		return
	}

	resolved = resolved.WithFallbackRunType()
	if _, err = s.queryToAbsPath(resolved); err == nil {
		// path with run type ANY and role name "any" exists, return it
		return
	}

	return nil, fmt.Errorf("could not resolve configuration path %s", query.AbsoluteRaw())
}
