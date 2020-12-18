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

package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/configuration/cfgbackend"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/flosch/pongo2/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "confsys")

type Service struct {
	src cfgbackend.Source
}

/* Expected structure:
/o2/aliecs/
{
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
	consulKey = strings.TrimLeft(key, componentcfg.SEPARATOR)
	return
}

func newService(uri string) (svc *Service, err error) {
	var src cfgbackend.Source
	src, err = cfgbackend.NewSource(uri)
	return &Service{src: src}, err
}

func (s *Service) NewDefaultRepo(defaultRepo string) error {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Put(filepath.Join(s.getConsulRuntimePrefix(),"default_repo"), defaultRepo)
	} else {
		data := []byte(defaultRepo)
		return ioutil.WriteFile(filepath.Join(s.GetReposPath(),"default_repo"), data, 0644)
	}
}

func (s *Service) GetDefaultRepo() (defaultRepo string, err error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Get(filepath.Join(s.getConsulRuntimePrefix(),"default_repo"))
	} else {
		var defaultRepoData []byte
		defaultRepoData, err = ioutil.ReadFile(filepath.Join(s.GetReposPath(),"default_repo"))
		if err != nil {
			return
		}
		defaultRepo = strings.TrimSuffix(string(defaultRepoData), "\n")
		return
	}
}

func (s *Service) NewDefaultRevision(defaultRevision string) error {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Put(filepath.Join(s.getConsulRuntimePrefix(),"default_revision"), defaultRevision)
	} else {
		data := []byte(defaultRevision)
		return ioutil.WriteFile(filepath.Join(s.GetReposPath(),"default_revision"), data, 0644)
	}
}

func (s *Service) GetDefaultRevision() (defaultRevision string, err error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Get(filepath.Join(s.getConsulRuntimePrefix(),"default_revision"))
	} else {
		var defaultRevisionData []byte
		defaultRevisionData, err = ioutil.ReadFile(filepath.Join(s.GetReposPath(),"default_revision"))
		if err != nil {
			return
		}
		defaultRevision = strings.TrimSuffix(string(defaultRevisionData), "\n")
		return
	}
}

func (s *Service) GetRepoDefaultRevisions() (map[string]string, error) {
	var defaultRevisions map[string]string
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		data, err := cSrc.Get(filepath.Join(s.getConsulRuntimePrefix(),"default_revisions"))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(data), &defaultRevisions)
		if err != nil {
			return nil, err
		}
	} else {
		defaultRevisionData, err := ioutil.ReadFile(filepath.Join(s.GetReposPath(),"default_revisions.json"))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(defaultRevisionData, &defaultRevisions)
	}
	return defaultRevisions, nil
}

func (s *Service) SetRepoDefaultRevisions(defaultRevisions map[string]string) error {
	data, err := json.MarshalIndent(defaultRevisions, "", "    ")
	if err != nil {
		return err
	}

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		err = cSrc.Put(filepath.Join(s.getConsulRuntimePrefix(),"default_revisions"), string(data))
	} else {
		err = ioutil.WriteFile(filepath.Join(s.GetReposPath(),"default_revisions.json"), data, 0644)
	}
	return err
}

func (s *Service) NewRunNumber() (runNumber uint32, err error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.GetNextUInt32(filepath.Join(s.getConsulRuntimePrefix(),"run_number"))
	} else {
		// Unsafe check-and-set, only for file backend
		var rnf string
		rnf = filepath.Join(viper.GetString("coreWorkingDir"), "runcounter.txt")
		if _, err = os.Stat(rnf); os.IsNotExist(err) {
			err = ioutil.WriteFile(rnf, []byte("0"), 0644)
			if err != nil {
				return
			}
		}
		var raw []byte
		raw, err = ioutil.ReadFile(rnf)
		if err != nil {
			return
		}
		var rn64 uint64
		rn64, err = strconv.ParseUint(string(raw[:]), 10, 32)
		if err != nil {
			return
		}
		runNumber = uint32(rn64)
		runNumber++
		raw = []byte(strconv.FormatUint(uint64(runNumber), 10))
		err = ioutil.WriteFile(rnf, raw, 0)
		return
	}
}

// maybe this one shouldn't exist at all, because vars should get inserted
// response: but not all of them! some vars will likely only get parsed at deployment time i.e. right
// before pushing TaskInfos
func (s *Service) GetDefaults() map[string]string {
	smap := s.getStringMap(filepath.Join(s.getConsulRuntimePrefix(),"defaults"))

	// Fill in some global constants we want to make available everywhere
	globalConfigurationUri := viper.GetString("globalConfigurationUri")
	smap["consul_base_uri"] = globalConfigurationUri
	consulUrl, err := url.ParseRequestURI(globalConfigurationUri)
	if err == nil {
		smap["consul_hostname"] = consulUrl.Hostname()
		smap["consul_port"] = consulUrl.Port()
		smap["consul_endpoint"] = consulUrl.Host
	} else {
		log.WithField("globalConfigurationUri", globalConfigurationUri).
			Warn("cannot parse global configuration endpoint")
	}
	smap["framework_id"], _ = s.GetMesosFID()
	smap["core_hostname"], _ = os.Hostname()
	return smap
}

func (s *Service) GetVars() map[string]string {
	return s.getStringMap(filepath.Join(s.getConsulRuntimePrefix(),"vars"))
}

// Or maybe even "RefreshConfig" which will refresh all the things that happen to be runtime-refreshable
func (s *Service) RefreshRepositories() {
	panic("not implemented yet")
}

// Returns a YAML file OR even a structure made of Roles or Nodes with:
// import() functions already computed and resolved
// vars inserted
func (s *Service) GenerateWorkflowDescriptor(wfPath string, vars map[string]string /*vars from cli/gui*/) string {
	panic("not implemented yet")
}

// Persist Mesos Framework ID by saving to Consul, or to a local file.
func (s *Service) SetMesosFID(fidValue string) error {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Put(filepath.Join(s.getConsulRuntimePrefix(),"mesos_fid"), fidValue)
	} else {
		data := []byte(fidValue)
		return ioutil.WriteFile(filepath.Join(viper.GetString("coreWorkingDir"), "mesos_fid.txt"), data, 0644)
	}
}

// Retrieve Mesos Framework ID from Consul, or local file.
func (s *Service) GetMesosFID() (fidValue string, err error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Get(filepath.Join(s.getConsulRuntimePrefix(),"mesos_fid"))
	} else {
		var byteFidValue []byte
		byteFidValue, err = ioutil.ReadFile(filepath.Join(viper.GetString("coreWorkingDir"), "mesos_fid.txt"))
		if err != nil {
			return
		}
		fidValue = strings.TrimSuffix(string(byteFidValue), "/n")
		return
	}
}

func (s *Service) GetReposPath() string {
	return filepath.Join(viper.GetString("coreWorkingDir"), "repos")
}

func (s *Service) GetComponentConfiguration(query *componentcfg.Query) (payload string, err error) {
	if query == nil {
		return
	}

	var timestamp string

	if len(query.Timestamp) == 0 {
		keyPrefix := query.AbsoluteWithoutTimestamp()
		if s.src.IsDir(keyPrefix) {
			var keys []string
			keys, err = s.src.GetKeysByPrefix(keyPrefix)
			if err != nil {
				return
			}
			timestamp, err = componentcfg.GetLatestTimestamp(keys, query)
			if err != nil {
				return
			}
		}
	}
	absKey := query.AbsoluteWithoutTimestamp() + componentcfg.SEPARATOR + timestamp
	if exists, _ := s.src.Exists(absKey); exists && len(timestamp) > 0 {
		payload, err = s.src.Get(absKey)
	} else {
		// falling back to timestampless configuration
		absKey = query.AbsoluteWithoutTimestamp()
		payload, err = s.src.Get(absKey)
	}
	return
}

func (s *Service) GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error) {
	path := query.Path()

	// We need to decompose the requested GetConfig path into prefix and suffix,
	// with the last / as separator (any timestamp if present stays part of the
	// suffix).
	// This is done to allow internal references between template snippets
	// within the same Consul directory.
	indexOfLastSeparator := strings.LastIndex(path, "/")
	var basePath string
	shortPath := path
	if indexOfLastSeparator != -1 {
		basePath = path[:indexOfLastSeparator]
		shortPath = path[indexOfLastSeparator+1:]
	}

	// We declare a TemplateSet, with a custom TemplateLoader.
	// Our ConsulTemplateLoader takes control of the FromFile code path
	// in pongo2, effectively adding support for Consul as file-like
	// backend.
	tplSet := pongo2.NewSet("", template.NewConsulTemplateLoader(s, basePath))
	var tpl *pongo2.Template
	tpl, err = tplSet.FromFile(shortPath)

	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error()), err
	}

	bindings := make(map[string]interface{})
	for k, v := range varStack {
		bindings[k] = v
	}

	// Add custom functions to bindings:
	funcMap := template.MakeStrOperationFuncMap()
	for k, v := range funcMap {
		bindings[k] = v
	}

	payload, err = tpl.Execute(bindings)
	return
}

func (s *Service) RawGetRecursive(path string) (string, error) {
	cfgDump, err := s.src.GetRecursive(path)
	if err != nil {
		log.WithError(err).Fatal("cannot retrieve configuration")
		return "", err
	}
	cfgBytes, err := json.MarshalIndent(cfgDump, "", "\t")
	if err != nil {
		log.WithError(err).Fatal("cannot marshal configuration dump")
		return "", err
	}
	return string(cfgBytes[:]), nil
}

func (s *Service) getConsulRuntimePrefix() string {
	// FIXME: this should not be hardcoded
	return "o2/aliecs"
}
