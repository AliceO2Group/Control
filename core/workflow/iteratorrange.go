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

package workflow

import (
	"encoding/json"
	"strconv"
	texttemplate "text/template"

	"github.com/AliceO2Group/Control/configuration/template"
	"github.com/AliceO2Group/Control/configuration/the"
)

type iteratorRange interface {
	copyable
	GetRange(varStack map[string]string) (ran []string, err error)
	GetVar() string
}

type iteratorRangeExpr struct {
	Range       string                  `yaml:"range"`
	Var         string                  `yaml:"var"`
}

func (f *iteratorRangeExpr) copy() copyable {
	itrCopy := iteratorRangeExpr{
		Range: f.Range,
		Var:   f.Var,
	}
	return &itrCopy
}

func (f *iteratorRangeExpr) GetRange(varStack map[string]string) (ran []string, err error) {
	rangeObj := make([]string, 0)

	fields := template.Fields{
		template.WrapPointer(&f.Range),
	}
	err = fields.Execute(the.ConfSvc(), "", varStack, make(map[string]interface{}), make(map[string]texttemplate.Template))
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(f.Range), &rangeObj)
	ran = rangeObj
	return
}

func (f *iteratorRangeExpr) GetVar() string {
	return f.Var
}


type iteratorRangeFor struct {
	Begin       string                  `yaml:"begin"`
	End         string                  `yaml:"end"`
	Var         string                  `yaml:"var"`
}

func (f *iteratorRangeFor) copy() copyable {
	itrCopy := iteratorRangeFor{
		Begin: f.Begin,
		End:   f.End,
		Var:   f.Var,
	}
	return &itrCopy
}

func (f *iteratorRangeFor) GetRange(varStack map[string]string) (ran []string, err error) {
	ran = make([]string, 0)

	fields := template.Fields{
		template.WrapPointer(&f.Begin),
		template.WrapPointer(&f.End),
	}
	err = fields.Execute(the.ConfSvc(), "", varStack, make(map[string]interface{}), make(map[string]texttemplate.Template))
	if err != nil {
		return
	}

	var begin, end int
	begin, err = strconv.Atoi(f.Begin)
	if err != nil {
		return
	}
	end, err = strconv.Atoi(f.End)
	if err != nil {
		return
	}

	for j := begin; j <= end; j++ {
		ran = append(ran, strconv.Itoa(j))
	}
	return
}

func (f *iteratorRangeFor) GetVar() string {
	return f.Var
}
