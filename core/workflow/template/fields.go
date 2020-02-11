/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018-2020 CERN and copyright holders of ALICE O².
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

package template

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/valyala/fasttemplate"
)

type Fields []Field

func (fields Fields) Execute(parentPath string, varStack map[string]string, stringTemplateCache map[string]template.Template) (err error) {
	environment := make(map[string]interface{}, len(varStack))
	for k, v := range varStack {
		environment[k] = v
	}

	for _, str := range fields {
		buf := new(bytes.Buffer)
		// FIXME: the line below implements the cache
		//tmpl, ok := stringTemplateCache[*str]
		var tmpl *fasttemplate.Template // dummy
		ok := false                     // dummy
		if !ok {
			tmpl, err = fasttemplate.NewTemplate(str.Get(), "{{", "}}")
			if err != nil {
				return
			}
		}
		_, err = tmpl.ExecuteFunc(buf, func(w io.Writer, tag string) (i int, err error) {
			var(
				program *vm.Program
				rawOutput interface{}
			)
			program, err = expr.Compile(tag)
			if err != nil {
				return
			}
			rawOutput, err = expr.Run(program, environment)
			if err != nil {
				return
			}

			return w.Write([]byte(fmt.Sprintf("%v", rawOutput)))
		})
		if err != nil {
			return
		}
		str.Set(buf.String())
	}
	return
}