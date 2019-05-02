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

package confsys

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/configuration"
)


type Service struct {
	src configuration.Source
}

/* Expected structure:
/o2/aliecs/
{
	run_number_file: "/path/to/rn/file",
        - or -
	run_number: 47102,

	settings: {
		log_level: "DEBUG"
	},
	global_vars: {
		n_flps: 42
	},
	repositories: {
		base: {
			url: "https://gitlab.cern.ch/foo/bar",
			user: "gituser",
			pass: "gitpass"
		},
		extra: {...}
	}
}

*/
func formatKey(key string) (consulKey string) {
	// Trim leading slashes
	consulKey = strings.TrimLeft(key, "/")
	return
}

func NewService(uri string) (svc *Service, err error) {
	var src configuration.Source
	src, err = configuration.NewSource(uri)
	return &Service{src: src}, err
}

func (s *Service) NewRunNumber() (runNumber uint64, err error) {
	if cSrc, ok := s.src.(*configuration.ConsulSource); ok {
		return cSrc.GetNextUInt64("o2/control/run_number")
	} else {
		// Unsafe check-and-set, only for file backend
		var rnf string
		rnf, err = s.src.Get("o2/control/run_number_file")
		if err != nil {
			return
		}
		var raw []byte
		raw, err = ioutil.ReadFile(rnf)
		if err != nil {
			return
		}
		runNumber, err = strconv.ParseUint(string(raw[:]), 10, 64)
		if err != nil {
			return
		}
		runNumber++
		raw = []byte(strconv.FormatUint(runNumber, 10))
		err = ioutil.WriteFile(rnf, raw, 0)
		return
	}
}

func (s *Service) GetROSource() configuration.ROSource {
	return s.src
}

// maybe this one shouldn't exist at all, because vars should get inserted
// response: but not all of them! some vars will likely only get parsed at deployment time i.e. right
//    before pushing TaskInfos
func (s *Service) GetVars() map[string]string {
	//FIXME: implement
	return nil
}

// Or maybe even "RefreshConfig" which will refresh all the things that happen to be runtime-refreshable
func (s *Service) RefreshRepositories() {
	panic("not implemented yet")
}

// Returns a YAML file OR even a structure made of Roles or Nodes with:
// import() functions already computed and resolved
// vars inserted (todo: figure out which vars get parsed when and where)
func (s *Service) GenerateWorkflowDescriptor(wfPath string, vars map[string]string /*vars from cli/gui*/) string {
	panic("not implemented yet")
}