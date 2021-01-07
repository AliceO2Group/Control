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
	"encoding/json"
	"errors"
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

func NewService(uri string) (svc *Service, err error) {
	var src cfgbackend.Source
	src, err = cfgbackend.NewSource(uri)
	return &Service{
		src: src,
	}, err
}

func (s *Service) NewRunNumber() (runNumber uint32, err error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.GetNextUInt32(filepath.Join(getConsulRuntimePrefix(),"run_number"))
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
	smap := s.getStringMap(filepath.Join(getConsulRuntimePrefix(),"defaults"))

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
	smap["framework_id"], _ = s.GetRuntimeEntry("aliecs", "mesos_fid")
	smap["core_hostname"], _ = os.Hostname()
	return smap
}

func (s *Service) GetVars() map[string]string {
	return s.getStringMap(filepath.Join(getConsulRuntimePrefix(),"vars"))
}

// Returns a YAML file OR even a structure made of Roles or Nodes with:
// import() functions already computed and resolved
// vars inserted
func (s *Service) GenerateWorkflowDescriptor(wfPath string, vars map[string]string /*vars from cli/gui*/) string {
	panic("not implemented yet")
}

// TODO: remove these, replaced by 2 calls to Get/SetRuntimeEntry in task/manager.go
//
//// Persist Mesos Framework ID by saving to Consul, or to a local file.
//func (s *Service) SetMesosFID(fidValue string) error {
//	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
//		return cSrc.Put(filepath.Join(getConsulRuntimePrefix(),"mesos_fid"), fidValue)
//	} else {
//		data := []byte(fidValue)
//		return ioutil.WriteFile(filepath.Join(viper.GetString("workingDir"), "mesos_fid.txt"), data, 0644)
//	}
//}
//
//// Retrieve Mesos Framework ID from Consul, or local file.
//func (s *Service) GetMesosFID() (fidValue string, err error) {
//	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
//		return cSrc.Get(filepath.Join(getConsulRuntimePrefix(),"mesos_fid"))
//	} else {
//		var byteFidValue []byte
//		byteFidValue, err = ioutil.ReadFile(filepath.Join(viper.GetString("workingDir"), "mesos_fid.txt"))
//		if err != nil {
//			return
//		}
//		fidValue = strings.TrimSuffix(string(byteFidValue), "/n")
//		return
//	}
//}

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

func (s *Service) GetRuntimeEntry(component string, key string) (string, error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Get(filepath.Join(getConsulRuntimePrefix(), component, key))
	} else {
		return "", errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) SetRuntimeEntry(component string, key string, value string) error {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Put(filepath.Join(getConsulRuntimePrefix(), component, key), value)
	} else {
		return errors.New("runtime KV not supported with file backend")
	}
}

func getConsulRuntimePrefix() string {
	// FIXME: this should not be hardcoded
	return "o2/aliecs"
}
