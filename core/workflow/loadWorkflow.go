/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
 * Author: Ayaan Zaidi <azaidi@cern.ch>
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
	"fmt"

	"github.com/k0kubun/pp"
	"gopkg.in/yaml.v2"
)

func LoadWorkflow(input []byte) (i *iteratorRole, a *aggregatorRole, err error) {
	// Unmarshal given YAML to map[string]interface{}
	var yamlData map[string]interface{}
	err = yaml.Unmarshal(input, &yamlData)
	if err != nil {
		log.Fatal(err)
	}
	isFor := false

	if _, ok := yamlData["roles"]; ok {
		for idx := range yamlData["roles"].([]interface{}) {
			switch v := yamlData["roles"].([]interface{})[idx].(type) {
			case map[interface{}]interface{}:
				if _, ok := v["for"]; ok {
					isFor = ok
				}
			}
		}
	}

	if isFor {
		err := yaml.Unmarshal(input, &i)
		if err != nil {
			return i, a, fmt.Errorf("%w", err)
		}
		return i, a, nil
	} else {
		err := yaml.Unmarshal(input, &a)
		if err != nil {
			log.Fatal(err)
		}
		return i, a, nil
	}

	return
}

func UnmarshalIterator(input []byte) (output *iteratorRole, err error) {
	err = yaml.Unmarshal(input, &output)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = pp.Println(&output)
	return
}
