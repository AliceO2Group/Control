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

package workflow

import (
	"text/template"
	"errors"
	"github.com/jinzhu/copier"
)

type includeTemplate struct {
	includeRole
	stringTemplates map[string]template.Template `yaml:"-,omitempty"`
}

func (tt *includeTemplate) copy() copyable {
	rCopy := includeTemplate{
		includeRole: *tt.includeRole.copy().(*includeRole),
	}
	_ = copier.Copy(&rCopy.stringTemplates, &tt.stringTemplates)
	return &rCopy
}

func (tt *includeTemplate) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _includeTemplate includeTemplate
	aux := _includeTemplate{}
	err = unmarshal(&aux)
	if err != nil {
		return
	}
	aux.stringTemplates = make(map[string]template.Template)

	/* template cache builder
	tmpl := template.New(aux.GetPath())

	// Fields to parse as templates:
	for _, str := range []string{
		aux.LoadTaskClass,
		aux.Name,
	} {
		var tempTmpl *template.Template
		tempTmpl, err = tmpl.Parse(str)
		if err != nil {
			return
		}
		aux.stringTemplates[str] = *tempTmpl
	}
	*/

	*tt = includeTemplate(aux)
	return
}

func (tt *includeTemplate) generateRole(localVars map[string]string) (c Role, err error) {
	if tt == nil {
		return nil, errors.New("cannot generate from nil sender")
	}

	// See note for aggregatorTemplate.generateRole
	tr := *tt.includeRole.copy().(*includeRole)
	for k, v := range localVars {
		tr.Locals[k] = v
	}

	c = &tr
	return
}