/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package workflow

import (
	"errors"
	"github.com/jinzhu/copier"
	"text/template"
)

type aggregatorTemplate struct {
	aggregatorRole
	// FIXME: replace ↓ with a map[string]*expr.vm.Program
	stringTemplates map[string]template.Template `yaml:"-,omitempty"`
}

func (at *aggregatorTemplate) copy() copyable {
	rCopy := aggregatorTemplate{
		aggregatorRole: *at.aggregatorRole.copy().(*aggregatorRole),
	}
	_ = copier.Copy(&rCopy.stringTemplates, &at.stringTemplates)
	return &rCopy
}

func (at *aggregatorTemplate) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _aggregatorTemplate aggregatorTemplate
	aux := _aggregatorTemplate{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}
	aux.stringTemplates = make(map[string]template.Template)

	/* template cache builder
	tmpl := template.New(aux.GetPath())
	// Fields to parse as templates:
	templatableFields := []string{
		aux.Name,
	}

	// FIXME: actually use these templates
	for _, str := range templatableFields {
		var tempTmpl *template.Template
		tempTmpl, err = tmpl.Parse(str)
		if err != nil {
			return
		}
		aux.stringTemplates[str] = *tempTmpl
	}
	*/

	*at = aggregatorTemplate(aux)
	return
}

func (at *aggregatorTemplate) generateRole(localVars map[string]string) (c Role, err error) {
	if at == nil {
		return nil, errors.New("cannot generate from nil sender")
	}

	// aggregatorTemplate.UnmarshalYAML exists, and it fills out at.stringTemplates.
	// at.stringTemplates contains cached compiled text/template.Template instances:
	// these will generate strings for us on request.
	// In this method:
	// 1) create new instance of aggregatorRole as copy
	// 2) push iterator-provided vars into the local Vars map

	// 1
	ar := *at.aggregatorRole.copy().(*aggregatorRole)
	// 2)
	for k, v := range localVars {
		ar.Locals[k] = v
	}

	c = &ar
	return
}
