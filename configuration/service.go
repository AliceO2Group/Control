/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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
	"github.com/AliceO2Group/Control/configuration/componentcfg"
)

type RuntimeService interface {
	GetRuntimeEntry(component string, key string) (string, error)
	GetRuntimeEntries(component string) (map[string]string, error)
	SetRuntimeEntry(component string, key string, value string) error
	ListRuntimeEntries(component string) ([]string, error)
}

type Service interface {
	RuntimeService
	NewRunNumber() (runNumber uint32, err error)
	GetDefaults() map[string]string
	GetVars() map[string]string
	InvalidateComponentTemplateCache()
	GetComponentConfiguration(query *componentcfg.Query) (payload string, err error)
	GetComponentConfigurationWithLastIndex(query *componentcfg.Query) (payload string, lastIndex uint64, err error)
	GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error)

	// getAll == false should skip TRG in the list
	ListDetectors(getAll bool) (detectors []string, err error)
	GetHostInventory(detector string) (hosts []string, err error)
	GetDetectorsInventory() (inventory map[string][]string, err error)
	ListComponents() (components []string, err error)
	ListComponentEntries(query *componentcfg.EntriesQuery) (entries []string, err error)
	ResolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error)

	ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool) (existingComponentUpdated bool, existingEntryUpdated bool, err error)

	GetDetectorForHost(hostname string) (string, error)
	GetDetectorsForHosts(hosts []string) ([]string, error)
	GetCRUCardsForHost(hostname string) (string, error)
	GetEndpointsForCRUCard(hostname, cardSerial string) (string, error)

	RawGetRecursive(path string) (string, error)
}
