/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019-2021 CERN and copyright holders of ALICE O².
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
	"sort"
	"strconv"
	"strings"
	"time"

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
		return cSrc.GetNextUInt32(filepath.Join(getConsulRuntimePrefix(), "run_number"))
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
	smap := s.getStringMap(filepath.Join(getAliECSRuntimePrefix(), "defaults"))

	// Fill in some global constants we want to make available everywhere
	var configUri string
	if viper.IsSet("config_endpoint") { //coconut
		configUri = viper.GetString("config_endpoint")
	} else if viper.IsSet("globalConfigurationUri") { //core
		configUri = viper.GetString("globalConfigurationUri")
	} else { //apricot
		configUri = viper.GetString("backendUri")
	}

	smap["consul_base_uri"] = configUri
	consulUrl, err := url.ParseRequestURI(configUri)
	if err == nil {
		smap["consul_hostname"] = consulUrl.Hostname()
		smap["consul_port"] = consulUrl.Port()
		smap["consul_endpoint"] = consulUrl.Host
	} else {
		log.WithField("globalConfigurationUri", configUri).
			Warn("cannot parse global configuration endpoint")
	}
	smap["framework_id"], _ = s.GetRuntimeEntry("aliecs", "mesos_fid")
	smap["core_hostname"], _ = os.Hostname()
	return smap
}

func (s *Service) GetHostInventory() (hosts []string, err error) {
	keys, err := s.src.GetKeysByPrefix("o2/hardware/flps/")
	if err != nil {
		log.WithError(err).Fatal("Error, could not retrieve host list.")
		return []string{""}, err
	}
	i := 0
	hosts = make([]string, len(keys))
	for _, key := range keys {
		hostTrimed := strings.TrimPrefix(key, "o2/hardware/flps/")
		hostname := strings.Split(hostTrimed, "/")
		hosts[i] = hostname[0]
		i++
	}
	return hosts, err
}

func (s *Service) GetVars() map[string]string {
	return s.getStringMap(filepath.Join(getAliECSRuntimePrefix(), "vars"))
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
	} else {
		timestamp = query.Timestamp
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

func (s *Service) GetDetectorForHost(hostname string) (string, error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		keys, err := cSrc.GetKeysByPrefix(filepath.Join("o2/hardware", "detectors"))
		if err != nil {
			return "", err
		}
		for _, key := range keys {
			// key example: o2/hardware/detectors/TST/flps/some-hostname/
			splitKey := strings.Split(key, "/")
			if len(splitKey) == 7 {
				if splitKey[5] == hostname {
					return splitKey[3], nil
				}
			}
		}
		return "", fmt.Errorf("detector not found for host %s", hostname)
	} else {
		return "", errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) GetCRUCardsForHost(hostname string) (string, error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		var cards map[string]Cards
		var serials []string
		cfgCards, err := cSrc.Get(filepath.Join("o2/hardware", "flps", hostname, "cards"))
		if err != nil {
			return "", err
		}
		json.Unmarshal([]byte(cfgCards), &cards)
	    unique := make(map[string]bool)
		for _, card := range cards  {
			if _, value := unique[card.Serial]; !value {
            	unique[card.Serial] = true
            	serials = append(serials, card.Serial)
        	}
		}
		bytes, err := json.Marshal(serials)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	} else {
		return "", errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) GetEndpointsForCRUCard(hostname, cardSerial string) (string, error) {
	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		var cards map[string]Cards
		var endpoints string
		cfgCards, err := cSrc.Get(filepath.Join("o2/hardware", "flps", hostname, "cards"))
		if err != nil {
			return "", err
		}
		json.Unmarshal([]byte(cfgCards), &cards)
		for _, card := range cards  {
			if card.Serial == cardSerial {
				endpoints = endpoints + card.Endpoint + " "
			}
		}
		return endpoints, nil
	} else {
		return "", errors.New("runtime KV not supported with file backend")
	}
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

func (s *Service) ListComponents() (components []string, err error) {
	keyPrefix := componentcfg.ConfigComponentsPath
	var keys []string
	keys, err = s.src.GetKeysByPrefix(keyPrefix)
	if err != nil {
		return
	}
	componentSet := make(map[string]struct{})
	for _, key := range keys {
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, componentcfg.SEPARATOR)
		// Criterion for being a component:
		// length of parts == 2 because of trailing slash in Consul output for folders
		// part[1] must be len=0, otherwise it's an actual entry within the component and not a trailing slash
		if len(componentParts) != 2 || len(componentParts[1]) != 0 {
			continue
		}
		component := componentParts[0]
		componentSet[component] = struct{}{}
	}
	components = make([]string, len(componentSet))
	i := 0
	for component, _ := range componentSet {
		components[i] = component
		i++
	}
	return
}

func formatComponentEntriesList(keys []string, keyPrefix string, showTimestamp bool) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, errors.New("no keys found")
	}

	var components sort.StringSlice

	// map of key to timestamp
	componentsSet := make(map[string]string)

	for _, key := range keys {
		// The input is assumed to be absolute paths, so we must trim the prefix.
		// The prefix includes the component name, e.g. o2/components/readout
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, "/")

		var componentTimestamp string

		// The component name is already stripped as part of the keyPrefix.
		// len(ANY/any/entry[/timestamp]) is 4, therefore ↓
		if len(componentParts) == 3 {
			// 1st acceptable case: single untimestamped entry
			if len(componentParts[len(componentParts)-1]) == 0 { // means this is a folder key with trailing slash "ANY/any/"
				continue
			}

			componentTimestamp = "" // we're sure this path cannot contain a timestamp
			componentsSet[componentsFullName] = ""
		} else if len(componentParts) == 4 {
			// A 5-len componentParts could be a timestamped entry, or a folder
			// in the latter case, the final component is an empty string, because
			// the full path has a trailing slash.
			// For this reason, we have 2 cases: showTimestamp=true or false
			// If false, we only need to pick 5-len folders (in addition to 4-len
			// entries).
			// If true, we must pick all true 5-len entries in order to compare them
			// & pick the newest (in addition to, as usual, 4-len ones).
			componentTimestamp = componentParts[len(componentParts)-1]
			componentsFullName = strings.TrimSuffix(componentsFullName, componentcfg.SEPARATOR+componentTimestamp)
			if !showTimestamp {
				componentsSet[componentsFullName] = ""
			} else {
				// if we *do* need to compare timestamps to find the latest
				if len(componentTimestamp) == 0 { // means this is a folder key with trailing slash "component/ANY/any/entry/"
					continue
				}
				if strings.Compare(componentsSet[componentsFullName], componentTimestamp) < 0 {
					componentsSet[componentsFullName] = componentTimestamp
				}
			}
		} else {
			continue
		}
	}

	for entryKey, entryTimestamp := range componentsSet {
		if showTimestamp {
			if len(entryTimestamp) == 0 {
				components = append(components, entryKey)
			} else {
				components = append(components, entryKey+"@"+entryTimestamp)
			}
		} else {
			components = append(components, entryKey)
		}
	}

	sort.Sort(components)
	return components, nil
}

func (s *Service) ListComponentEntries(query *componentcfg.EntriesQuery, showLatestTimestamp bool) (entries []string, err error) {
	keyPrefix := componentcfg.ConfigComponentsPath
	if query == nil {
		err = errors.New("bad query for ListComponentEntries")
		return
	}

	keyPrefix += query.Component + "/"

	var keys []string
	keys, err = s.src.GetKeysByPrefix(keyPrefix)
	if err != nil {
		return
	}

	entries, err = formatComponentEntriesList(keys, keyPrefix, showLatestTimestamp)
	if err != nil {
		return
	}

	return
}

func (s *Service) ListComponentEntryHistory(query *componentcfg.Query) (entries []string, err error) {
	if query == nil {
		return
	}

	fullKeyToQuery := query.AbsoluteWithoutTimestamp()
	var keys sort.StringSlice
	keys, err = s.src.GetKeysByPrefix(fullKeyToQuery)
	if err != nil {
		return
	}
	if len(keys) == 0 {
		err = errors.New("empty data returned from configuration backend")
		return
	}

	if len(query.EntryKey) == 0 {
		err = errors.New("history requested for empty entry name")
		return
	}

	// We trim the prefix + component
	keyPrefix := componentcfg.ConfigComponentsPath + query.Component + componentcfg.SEPARATOR
	for i := 0; i < len(keys); i++ {
		trimmed := strings.TrimPrefix(keys[i], keyPrefix)
		componentParts := strings.Split(trimmed, componentcfg.SEPARATOR)
		if len(componentParts) != 4 {
			// bad key!
			continue
		}
		keys[i] = componentParts[0] + componentcfg.SEPARATOR +
			componentParts[1] + componentcfg.SEPARATOR +
			componentParts[2] + "@" +
			componentParts[3]
	}

	sort.Sort(sort.Reverse(keys))
	entries = keys

	return
}

func (s *Service) ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool, useVersioning bool) (existingComponentUpdated bool, existingEntryUpdated bool, newTimestamp int64, err error) {
	if query == nil {
		return
	}

	var keys []string
	keys, err = s.src.GetKeysByPrefix("")
	if err != nil {
		return
	}

	components := componentcfg.GetComponentsMapFromKeysList(keys)

	componentExist := components[query.Component]
	if !componentExist && !newComponent {
		componentMsg := ""
		for key := range components {
			componentMsg += "\n- " + key
		}
		err = errors.New("component " + query.Component + " does not exist. " +
			"Available components in configuration database:" + componentMsg +
			"\nTo create a new component, use the new component parameter")
		return
	}
	if componentExist && newComponent {
		err = errors.New("invalid use of new component parameter: component " + query.Component + " already exists")
		return
	}

	entryExists := false
	if !newComponent {
		entriesMap := componentcfg.GetEntriesMapOfComponentFromKeysList(query.Component, query.RunType, query.RoleName, keys)
		entryExists = entriesMap[query.EntryKey]
	}

	// Temporary workaround to allow no-versioning
	var latestTimestamp string
	latestTimestamp, err = componentcfg.GetLatestTimestamp(keys, query)
	if err != nil {
		return
	}

	if entryExists {
		if (latestTimestamp != "0" && latestTimestamp != "") && !useVersioning {
			// If a timestamp already exists in the entry specified by the user, than it cannot be used
			err = errors.New("Specified entry: '" + query.EntryKey + "' already contains versioned items. Please " +
				"specify a different entry name")
			return
		}
		if (latestTimestamp == "0" || latestTimestamp == "") && useVersioning {
			// If a timestamp does not exist for specified entry but user wants versioning than an error is thrown
			err = errors.New("Specified entry: '" + query.EntryKey + "' already contains un-versioned items. Please " +
				"specify a different entry name")
			return
		}
	}

	timestamp := time.Now().Unix()
	fullKey := query.AbsoluteWithoutTimestamp()

	if useVersioning {
		fullKey += componentcfg.SEPARATOR + strconv.FormatInt(timestamp, 10)
	}

	err = s.src.Put(fullKey, payload)
	if err != nil {
		return
	}

	existingComponentUpdated = componentExist
	existingEntryUpdated = entryExists
	newTimestamp = timestamp
	return
}

func getConsulRuntimePrefix() string {
	// FIXME: this should not be hardcoded
	return "o2/runtime"
}

func getAliECSRuntimePrefix() string {
	return getConsulRuntimePrefix() + "/aliecs"
}
