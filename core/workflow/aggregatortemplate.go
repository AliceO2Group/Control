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
	"github.com/jinzhu/copier"
	"errors"
	"text/template"
	)

type aggregatorTemplate struct {
	aggregatorRole
	stringTemplates map[string]template.Template `yaml:"-,omitempty"`
}

func (at *aggregatorTemplate) copy() copyable {
	rCopy := aggregatorTemplate{
		aggregatorRole: *at.aggregatorRole.copy().(*aggregatorRole),
	}
	copier.Copy(&rCopy.stringTemplates, &at.stringTemplates)
	return &rCopy
}


func (at *aggregatorTemplate) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	type _aggregatorTemplate aggregatorTemplate
	role := _aggregatorTemplate{}
	err = unmarshal(&role)
	if err != nil {
		return
	}
	tmpl := template.New(role.GetPath())
	role.stringTemplates = make(map[string]template.Template)

	// Fields to parse as templates:
	for _, str := range []string{
		role.Name,
	} {
		var tempTmpl *template.Template
		tempTmpl, err = tmpl.Parse(str)
		if err != nil {
			return
		}
		role.stringTemplates[str] = *tempTmpl
	}

	*at = aggregatorTemplate(role)
	return
}

func (at *aggregatorTemplate) generateRole(t templateMap) (c Role, err error) {
	if at == nil {
		return nil, errors.New("cannot generate from nil sender")
	}

	// NOTE:
	// aggregatorTemplate.UnmarshalYAML exists, and it fills out at.stringTemplates.
	// at.stringTemplates contains cached compiled text/template.Template instances:
	// these will generate strings for us on request.
	// In this method:
	// 1) create new instance of aggregatorRole
	// 2a) for each field not to templatify, just make a deep copy
	// 2b) for each field to templatify, execute the template against the templateMap
	// 3) ???
	// 4) PROFIT!

	// 1 + 2a)
	ar := *at.aggregatorRole.copy().(*aggregatorRole)

	// 2b)
	tf := templateFields{&ar.Name}
	err = tf.execute(at.GetPath(), t, at.stringTemplates)
	if err != nil {
		return
	}

	// 3 + 4)
	c = &ar
	return
}