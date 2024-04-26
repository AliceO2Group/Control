/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2021 CERN and copyright holders of ALICE O².
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

package componentcfg

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
)

const (
	FALLBACK_RUNTYPE = apricotpb.RunType_ANY

	FALLBACK_ROLENAME = "any"
)

var (
	//                                          component        /RUNTYPE          /rolename             /entry                @timestamp
	inputFullRegex = regexp.MustCompile(`^([a-zA-Z0-9-_]+)(\/[A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_/]+){1}(\@[0-9]+)?$`)
	//                                          component        /RUNTYPE          /rolename
	inputEntriesRegex    = regexp.MustCompile(`^([a-zA-Z0-9-_]+)(\/[A-Z0-9-_]+){1}(\/[a-z-A-Z0-9-_]+){1}$`)
	inputParametersRegex = regexp.MustCompile(`^([a-zA-Z0-9-_]+=[a-zA-Z0-9-_,]+)(&[a-zA-Z0-9-_]+=[a-zA-Z0-9-_,"\[\]]+)*$`)
	E_BAD_KEY            = errors.New("bad component configuration key format")
)

func IsStringValidQueryPathWithOptionalTimestamp(input string) bool {
	return inputFullRegex.MatchString(input)
}
func IsStringValidEntriesQueryPath(input string) bool {
	return inputEntriesRegex.MatchString(input)
}
func IsStringValidQueryParameters(input string) bool {
	return inputParametersRegex.MatchString(input)
}

type EntriesQuery struct {
	Component string
	RunType   apricotpb.RunType
	RoleName  string
}

func NewEntriesQuery(path string) (p *EntriesQuery, err error) {
	p = &EntriesQuery{
		Component: "",
		RunType:   apricotpb.RunType_NULL,
		RoleName:  "",
	}
	path = strings.TrimSpace(path)
	if IsStringValidEntriesQueryPath(path) {
		// coconut conf list component/FLAVOR/rolename
		params := strings.Split(path, SEPARATOR)
		p.Component = params[0]
		// Convert FLAVOR to pb-provided enum
		typedRunType, ok := apricotpb.RunType_value[params[1]]
		if !ok {
			err = E_BAD_KEY
			return
		}
		p.RunType = apricotpb.RunType(typedRunType)
		p.RoleName = params[2]
	} else {
		err = E_BAD_KEY
		return
	}

	return p, nil
}

type Query struct {
	Component string
	RunType   apricotpb.RunType
	RoleName  string
	EntryKey  string
	Timestamp string
}

func NewQuery(path string) (p *Query, err error) {
	p = &Query{
		Component: "",
		RunType:   apricotpb.RunType_NULL,
		RoleName:  "",
		EntryKey:  "",
		Timestamp: "",
	}
	path = strings.TrimSpace(path)
	if IsStringValidQueryPathWithOptionalTimestamp(path) {

		matches := inputFullRegex.FindAllStringSubmatch(path, -1)

		if len(matches) != 1 {
			err = E_BAD_KEY
			return
		}
		captureGroups := matches[0][1:] // the first element is the full query, we don't need it
		if len(captureGroups) < 4 && len(captureGroups) > 5 {
			err = E_BAD_KEY
			return
		}
		p.Component = captureGroups[0]
		// Convert FLAVOR to pb-provided enum
		typedFlavor, ok := apricotpb.RunType_value[strings.TrimPrefix(captureGroups[1], "/")]
		if !ok {
			err = E_BAD_KEY
			return
		}
		p.RunType = apricotpb.RunType(typedFlavor)
		p.RoleName = strings.TrimPrefix(captureGroups[2], "/")
		p.EntryKey = strings.TrimPrefix(captureGroups[3], "/")
		p.Timestamp = strings.TrimPrefix(captureGroups[4], "@")
	} else {
		err = E_BAD_KEY
		return
	}

	return p, nil
}

func (p *Query) WithFallbackRunType() *Query {
	return &Query{
		Component: p.Component,
		RunType:   FALLBACK_RUNTYPE,
		RoleName:  p.RoleName,
		EntryKey:  p.EntryKey,
		Timestamp: p.Timestamp,
	}
}

func (p *Query) WithFallbackRoleName() *Query {
	return &Query{
		Component: p.Component,
		RunType:   p.RunType,
		RoleName:  FALLBACK_ROLENAME,
		EntryKey:  p.EntryKey,
		Timestamp: p.Timestamp,
	}
}

func (p *Query) Path() string {
	path := p.WithoutTimestamp()
	if len(p.Timestamp) > 0 {
		return path + "@" + p.Timestamp
	}
	return path
}

func (p *Query) Raw() string {
	path := p.WithoutTimestamp()
	if len(p.Timestamp) > 0 {
		return path + SEPARATOR + p.Timestamp
	}
	return path
}

func (p *Query) WithoutTimestamp() string {
	return p.Component + SEPARATOR + apricotpb.RunType_name[int32(p.RunType)] + SEPARATOR + p.RoleName + SEPARATOR + p.EntryKey
}

func (p *Query) AbsoluteRaw() string {
	return ConfigComponentsPath + p.Raw()
}

func (p *Query) AbsoluteWithoutTimestamp() string {
	return ConfigComponentsPath + p.WithoutTimestamp()
}

type QueryParameters struct {
	ProcessTemplates bool
	VarStack         map[string]string
}

func NewQueryParameters(parameters string) (p *QueryParameters, err error) {
	p = &QueryParameters{
		ProcessTemplates: false,
		VarStack:         make(map[string]string),
	}
	parameters = strings.TrimSpace(parameters)

	if !IsStringValidQueryParameters(parameters) {
		err = E_BAD_KEY
		return
	}
	keyValues, err := url.ParseQuery(parameters)
	if err != nil {
		return
	}
	// in our case, we support just one value per key, thus we map the returned keyValues accordingly
	for key, values := range keyValues {
		if len(values) != 1 {
			err = E_BAD_KEY
			return
		}
		if key == "process" {
			p.ProcessTemplates, err = strconv.ParseBool(values[0])
			if err != nil {
				return
			}
		} else {
			p.VarStack[key] = values[0]
		}
	}

	return p, nil
}
