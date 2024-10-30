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

type taskTemplate struct {
	taskRole
	stringTemplates map[string]template.Template `yaml:"-,omitempty"`
}

func (tt *taskTemplate) copy() copyable {
	rCopy := taskTemplate{
		taskRole: *tt.taskRole.copy().(*taskRole),
	}
	_ = copier.Copy(&rCopy.stringTemplates, &tt.stringTemplates)
	return &rCopy
}

func (tt *taskTemplate) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _taskTemplate taskTemplate
	aux := _taskTemplate{}
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

	*tt = taskTemplate(aux)
	return
}

func (tt *taskTemplate) generateRole(localVars map[string]string) (c Role, err error) {
	if tt == nil {
		return nil, errors.New("cannot generate from nil sender")
	}

	// See note for aggregatorTemplate.generateRole
	tr := *tt.taskRole.copy().(*taskRole)
	for k, v := range localVars {
		tr.Locals[k] = v
	}

	c = &tr
	return
}
