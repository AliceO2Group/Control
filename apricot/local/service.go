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
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/configuration/cfgbackend"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/flosch/pongo2/v6"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var log = logger.New(logrus.StandardLogger(), "confsys")

const inventoryKeyPrefix = "o2/hardware/"

type Service struct {
	src cfgbackend.Source

	templateSets   map[string]*pongo2.TemplateSet
	templateSetsMu sync.Mutex
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
	s.logMethod()

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
	smap["core_hostname"], _ = os.Hostname() // fixme: this might not be correct if apricot sits elsewhere than the core
	return smap
}

func (s *Service) ListDetectors(getAll bool) (detectors []string, err error) {
	s.logMethod()

	keyPrefix := inventoryKeyPrefix + "detectors/"
	var keys []string
	keys, err = s.src.GetKeysByPrefix(keyPrefix)
	if err != nil {
		log.WithError(err).Error("could not retrieve detectors")
		return []string{}, err
	}
	detectorSet := make(map[string]bool, 0)
	detectors = make([]string, 0)
	for _, key := range keys {
		detTrimmed := strings.TrimPrefix(key, keyPrefix)
		detname := strings.Split(detTrimmed, "/")
		if !getAll && detname[0] == "TRG" {
			continue
		}
		if _, ok := detectorSet[detname[0]]; !ok { // the detector name we found in the path isn't already accounted for
			detectorSet[detname[0]] = true
			detectors = append(detectors, detname[0])
		}
	}
	return detectors, err
}

func (s *Service) GetHostInventory(detector string) (hosts []string, err error) {
	s.logMethod()

	var keyPrefix string
	if detector != "" {
		keyPrefix = inventoryKeyPrefix + "detectors/" + detector + "/flps/"
	} else {
		keyPrefix = inventoryKeyPrefix + "flps/"
	}
	var keys []string
	keys, err = s.src.GetKeysByPrefix(keyPrefix)
	if err != nil {
		log.WithError(err).Error("could not retrieve host list")
		return []string{}, err
	}
	hostSet := make(map[string]bool, 0)
	hosts = make([]string, 0)
	for _, key := range keys {
		hostTrimmed := strings.TrimPrefix(key, keyPrefix)
		hostname := strings.Split(hostTrimmed, "/")
		if _, ok := hostSet[hostname[0]]; !ok {
			hostSet[hostname[0]] = true
			hosts = append(hosts, hostname[0])
		}
	}
	return hosts, err
}

func (s *Service) GetDetectorsInventory() (inventory map[string][]string, err error) {
	s.logMethod()

	inventory = map[string][]string{}
	detectors, err := s.ListDetectors(true)
	if err != nil {
		log.WithError(err).Error("could not retrieve detectors list")
		return nil, err
	}
	for _, detector := range detectors {
		hosts, err := s.GetHostInventory(detector)
		if err != nil {
			log.WithError(err).Error("could not retrieve hosts list")
			return nil, err
		}
		inventory[detector] = hosts
	}
	return inventory, err
}

func (s *Service) GetVars() map[string]string {
	s.logMethod()

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
	s.logMethod()

	var absolutePath string
	absolutePath, err = s.queryToAbsPath(query)
	if err != nil {
		return
	}
	payload, err = s.src.Get(absolutePath)
	return
}

func (s *Service) GetComponentConfigurationWithLastIndex(query *componentcfg.Query) (payload string, lastIndex uint64, err error) {
	s.logMethod()

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		var absolutePath string
		absolutePath, err = s.queryToAbsPath(query)
		if err != nil {
			return
		}
		payload, lastIndex, err = cSrc.GetWithLastIndex(absolutePath)
	} else {
		err = errors.New("component with last index not supported with file backend")
	}
	return
}

func (s *Service) GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error) {
	s.logMethod()

	path := query.Path()

	// We need to decompose the requested GetConfig path into prefix and suffix,
	// with the last / as separator.
	// This is done to allow internal references between template snippets
	// within the same Consul directory.
	indexOfLastSeparator := strings.LastIndex(path, "/")
	var basePath string
	shortPath := path
	if indexOfLastSeparator != -1 {
		basePath = path[:indexOfLastSeparator]
		shortPath = path[indexOfLastSeparator+1:]
	}

	// We get a TemplateSet, with a custom TemplateLoader. Depending on past events, a template set for this base path
	// might already exist in the service's template set cache map. We will then use the cache of this template set to
	// speed up future requests.
	// In order for resolution of short paths to work (i.e. entry name within a component/runtype/rolename directory),
	// we need to build one templateSet+templateLoader per base path.
	// Our ConsulTemplateLoader takes control of the FromFile code path in pongo2, effectively adding support for
	// Consul as file-like backend.
	tplSet := s.templateSetForBasePath(basePath)
	var tpl *pongo2.Template
	tpl, err = tplSet.FromCache(shortPath)

	if err != nil {
		return fmt.Sprintf("{\"error\":\"%s\"}", err.Error()), err
	}

	bindings := make(map[string]interface{})
	for k, v := range varStack {
		bindings[strings.TrimSpace(k)] = v
	}

	// Add custom functions to bindings:
	funcMap := template.MakeUtilFuncMap(varStack)
	for k, v := range funcMap {
		bindings[k] = v
	}

	payload, err = tpl.Execute(bindings)
	return
}

func (s *Service) ResolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error) {
	s.logMethod()

	resolved = &componentcfg.Query{}
	if query == nil {
		*resolved = *query
		return
	}
	resolved, err = s.resolveComponentQuery(query)
	return
}

func (s *Service) RawGetRecursive(path string) (string, error) {
	s.logMethod()

	cfgDump, err := s.src.GetRecursive(path)
	if err != nil {
		log.WithError(err).Error("cannot retrieve configuration")
		return "", err
	}
	cfgBytes, err := json.MarshalIndent(cfgDump, "", "\t")
	if err != nil {
		log.WithError(err).Error("cannot marshal configuration dump")
		return "", err
	}
	return string(cfgBytes[:]), nil
}

func (s *Service) GetDetectorForHost(hostname string) (string, error) {
	s.logMethod()

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

func (s *Service) GetDetectorsForHosts(hosts []string) ([]string, error) {
	s.logMethod()

	detectorMap := make(map[string]struct{})
	for _, host := range hosts {
		det, err := s.GetDetectorForHost(host)
		if err != nil {
			return []string{}, err
		}
		detectorMap[det] = struct{}{}
	}

	detectorSlice := make([]string, len(detectorMap))
	i := 0
	for k, _ := range detectorMap {
		detectorSlice[i] = k
		i++
	}
	sort.Strings(detectorSlice)
	return detectorSlice, nil
}

func (s *Service) GetCRUCardsForHost(hostname string) (string, error) {
	s.logMethod()

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		var cards map[string]Card
		var serials []string
		cfgCards, err := cSrc.Get(filepath.Join("o2/hardware", "flps", hostname, "cards"))
		if err != nil {
			return "", err
		}
		json.Unmarshal([]byte(cfgCards), &cards)
		unique := make(map[string]bool)
		for _, card := range cards {
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
	s.logMethod()

	log.WithPrefix("rpcserver").
		WithField("method", "GetEndpointsForCRUCard").
		WithField("level", infologger.IL_Devel).
		WithField("hostname", hostname).
		WithField("cardSerial", cardSerial).
		Debug("getting endpoints")

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		var cards map[string]Card
		var endpoints string
		cfgCards, err := cSrc.Get(filepath.Join("o2/hardware", "flps", hostname, "cards"))
		if err != nil {
			return "", err
		}
		err = json.Unmarshal([]byte(cfgCards), &cards)
		if err != nil {
			return "", err
		}
		for _, card := range cards {
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
	s.logMethod()

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Get(filepath.Join(getConsulRuntimePrefix(), component, key))
	} else {
		return "", errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) SetRuntimeEntry(component string, key string, value string) error {
	s.logMethod()

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		return cSrc.Put(filepath.Join(getConsulRuntimePrefix(), component, key), value)
	} else {
		return errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) GetRuntimeEntries(component string) (map[string]string, error) {
	s.logMethod()

	if keys, err := s.ListRuntimeEntries(component); err == nil {
		var keysErrors *multierror.Error
		entries := make(map[string]string)
		for _, key := range keys {
			if entry, err := s.GetRuntimeEntry(component, key); err == nil {
				entries[key] = entry
			} else {
				keysErrors = multierror.Append(keysErrors, err)
			}
		}
		return entries, keysErrors.ErrorOrNil()
	} else {
		return nil, err
	}

}

func (s *Service) ListRuntimeEntries(component string) ([]string, error) {
	s.logMethod()

	if cSrc, ok := s.src.(*cfgbackend.ConsulSource); ok {
		path := filepath.Join(getConsulRuntimePrefix(), component)
		keys, err := cSrc.GetKeysByPrefix(path)
		if err != nil {
			return nil, err
		}

		payload := make([]string, 0)
		for _, k := range keys {
			keySuffix := strings.TrimPrefix(k, path+"/")
			if keySuffix == "" {
				continue
			}
			split := strings.Split(keySuffix, componentcfg.SEPARATOR)
			var last string
			last = split[len(split)-1]
			if last == "" {
				continue
			} else {
				payload = append(payload, keySuffix)
			}
		}
		return payload, nil
	} else {
		return nil, errors.New("runtime KV not supported with file backend")
	}
}

func (s *Service) ListComponents() (components []string, err error) {
	s.logMethod()

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

func formatComponentEntriesList(keys []string, keyPrefix string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, errors.New("no keys found")
	}

	var components sort.StringSlice

	for _, key := range keys {
		// The input is assumed to be absolute paths, so we must trim the prefix.
		// The prefix includes the component name, e.g. o2/components/readout
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, "/")

		// The component name is already stripped as part of the keyPrefix.
		// len(ANY/any/entry) is least 3, therefore ↓
		if len(componentParts) >= 3 {
			// 1st acceptable case: single entry
			if len(componentParts[len(componentParts)-1]) == 0 { // means this is a folder key with trailing slash e.g. "ANY/any/"
				continue
			}
			components = append(components, componentsFullName)
		} else {
			continue
		}
	}

	sort.Sort(components)
	return components, nil
}

func (s *Service) ListComponentEntries(query *componentcfg.EntriesQuery) (entries []string, err error) {
	s.logMethod()

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

	entries, err = formatComponentEntriesList(keys, keyPrefix)
	if err != nil {
		return
	}

	return
}

func (s *Service) ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool) (existingComponentUpdated bool, existingEntryUpdated bool, err error) {
	s.logMethod()

	if query == nil {
		return
	}

	var keys []string
	// fixme: it looks like an overkill to get all the keys in config tree just to obtain a list of components in o2/components
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

	fullKey := query.AbsoluteRaw()

	err = s.src.Put(fullKey, payload)
	if err != nil {
		return
	}

	existingComponentUpdated = componentExist
	existingEntryUpdated = entryExists
	return
}

func getConsulRuntimePrefix() string {
	// FIXME: this should not be hardcoded
	return "o2/runtime"
}

func getAliECSRuntimePrefix() string {
	return getConsulRuntimePrefix() + "/aliecs"
}

func (s *Service) logMethod() {
	if !viper.GetBool("verbose") {
		return
	}
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("rpcserver").
		WithField("method", fun.Name()).
		WithField("level", infologger.IL_Devel).
		Debug("handling RPC request")
}

func (s *Service) templateSetForBasePath(basePath string) *pongo2.TemplateSet {
	s.templateSetsMu.Lock()
	defer s.templateSetsMu.Unlock()
	if s.templateSets == nil {
		s.templateSets = make(map[string]*pongo2.TemplateSet)
	}
	if _, ok := s.templateSets[basePath]; !ok {
		s.templateSets[basePath] = pongo2.NewSet(basePath, template.NewConsulTemplateLoader(s, basePath))
	}
	return s.templateSets[basePath]
}

func (s *Service) InvalidateComponentTemplateCache() {
	s.templateSetsMu.Lock()
	defer s.templateSetsMu.Unlock()

	// In principle we could also foreach templateSet call ClearCache(), but this is quicker and has the same effect
	s.templateSets = make(map[string]*pongo2.TemplateSet)
}
