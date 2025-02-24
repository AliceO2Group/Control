package cacheproxy

import (
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
)

// Implements a cache proxy Service for the configuration system.
// Only DetectorForHost/DetectorsForHosts are cached, all other calls are passed through.
type Service struct {
	base  configuration.Service
	cache svcCache
}

type svcCache struct {
	detectorsInventory map[string][]string
	detectorForHost    map[string]string
}

func NewService(base configuration.Service) (*Service, error) {
	svc := &Service{
		base: base,
		cache: svcCache{
			detectorsInventory: make(map[string][]string),
			detectorForHost:    make(map[string]string),
		},
	}

	var err error
	svc.cache.detectorsInventory, err = svc.base.GetDetectorsInventory()
	if err != nil {
		return nil, err
	}

	for det, hosts := range svc.cache.detectorsInventory {
		for _, host := range hosts {
			svc.cache.detectorForHost[host] = det
		}
	}

	return svc, nil
}

func (s Service) GetRuntimeEntry(component string, key string) (string, error) {
	return s.base.GetRuntimeEntry(component, key)
}

func (s Service) SetRuntimeEntry(component string, key string, value string) error {
	return s.base.SetRuntimeEntry(component, key, value)
}

func (s Service) GetRuntimeEntries(component string) (map[string]string, error) {
	return s.base.GetRuntimeEntries(component)
}

func (s Service) ListRuntimeEntries(component string) ([]string, error) {
	return s.base.ListRuntimeEntries(component)
}

func (s Service) NewRunNumber() (runNumber uint32, err error) {
	return s.base.NewRunNumber()
}

func (s Service) GetDefaults() map[string]string {
	return s.base.GetDefaults()
}

func (s Service) GetVars() map[string]string {
	return s.base.GetVars()
}

func (s Service) GetComponentConfiguration(query *componentcfg.Query) (payload string, err error) {
	return s.base.GetComponentConfiguration(query)
}

func (s Service) GetComponentConfigurationWithLastIndex(query *componentcfg.Query) (payload string, lastIndex uint64, err error) {
	return s.base.GetComponentConfigurationWithLastIndex(query)
}

func (s Service) GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error) {
	return s.base.GetAndProcessComponentConfiguration(query, varStack)
}

func (s Service) ListDetectors(getAll bool) (detectors []string, err error) {
	return s.base.ListDetectors(getAll)
}

func (s Service) GetHostInventory(detector string) (hosts []string, err error) {
	return s.base.GetHostInventory(detector)
}

func (s Service) GetDetectorsInventory() (inventory map[string][]string, err error) {
	return s.base.GetDetectorsInventory()
}

func (s Service) ListComponents() (components []string, err error) {
	return s.base.ListComponents()
}

func (s Service) ListComponentEntries(query *componentcfg.EntriesQuery) (entries []string, err error) {
	return s.base.ListComponentEntries(query)
}

func (s Service) ResolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error) {
	return s.base.ResolveComponentQuery(query)
}

func (s Service) ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool) (existingComponentUpdated bool, existingEntryUpdated bool, err error) {
	return s.base.ImportComponentConfiguration(query, payload, newComponent)
}

func (s Service) GetDetectorForHost(hostname string) (string, error) {
	det, ok := s.cache.detectorForHost[hostname]
	if !ok {
		return s.base.GetDetectorForHost(hostname)
	}
	return det, nil
}

func (s Service) GetDetectorsForHosts(hosts []string) ([]string, error) {
	detectors := make(map[string]struct{}, 0)
	for _, host := range hosts {
		det, ok := s.cache.detectorForHost[host]
		if !ok {
			return s.base.GetDetectorsForHosts(hosts)
		}
		detectors[det] = struct{}{}
	}
	detList := make([]string, 0, len(detectors))
	for det := range detectors {
		detList = append(detList, det)
	}
	return detList, nil
}

func (s Service) GetCRUCardsForHost(hostname string) ([]string, error) {
	return s.base.GetCRUCardsForHost(hostname)
}

func (s Service) GetEndpointsForCRUCard(hostname, cardSerial string) ([]string, error) {
	return s.base.GetEndpointsForCRUCard(hostname, cardSerial)
}

func (s Service) RawGetRecursive(path string) (string, error) {
	return s.base.RawGetRecursive(path)
}

func (s Service) InvalidateComponentTemplateCache() {
	s.base.InvalidateComponentTemplateCache()
}
