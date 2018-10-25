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
	"text/template"
	"errors"
	"github.com/jinzhu/copier"
)

type taskTemplate struct {
	taskRole
	stringTemplates map[string]template.Template `yaml:"-,omitempty"`
}

func (at *taskTemplate) copy() copyable {
	rCopy := taskTemplate{
		taskRole: *at.taskRole.copy().(*taskRole),
	}
	copier.Copy(&rCopy.stringTemplates, &at.stringTemplates)
	return &rCopy
}

func (tt *taskTemplate) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _taskTemplate taskTemplate
	role := _taskTemplate{}
	err = unmarshal(&role)
	if err != nil {
		return
	}
	tmpl := template.New(role.GetPath())
	role.stringTemplates = make(map[string]template.Template)

	// Fields to parse as templates:
	for _, str := range []string{
		role.LoadTaskClass,
		role.Name,
	} {
		var tempTmpl *template.Template
		tempTmpl, err = tmpl.Parse(str)
		if err != nil {
			return
		}
		role.stringTemplates[str] = *tempTmpl
	}

	*tt = taskTemplate(role)
	return
}

func (tt *taskTemplate) generateRole(t templateMap) (c Role, err error) {
	if tt == nil {
		return nil, errors.New("cannot generate from nil sender")
	}

	// See NOTE for aggregatorTemplate.generateRole
	tr := *tt.taskRole.copy().(*taskRole)

	tf := templateFields{&tr.Name, &tr.LoadTaskClass}
	err = tf.execute(tt.GetPath(), t, tt.stringTemplates)
	if err != nil {
		return
	}

	c = &tr
	return
}