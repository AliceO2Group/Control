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

package validate

import (
	"errors"
	"fmt"
	"os"

	"github.com/AliceO2Group/Control/walnut/schemata"
	"gopkg.in/yaml.v2"

	"github.com/xeipuuv/gojsonschema"
)

type inputYAML map[string]interface{}

func (m *inputYAML) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var data interface{}
	err = unmarshal(&data)
	if err != nil {
		return err
	}

	data = convert(data)

	return
}

// CheckSchema accepts YAML file and format then validate against the schema specified (either workflow or task)
func CheckSchema(rawYAML []byte, format string) (err error) {

	var dataFromYAML interface{}
	if err := yaml.Unmarshal(rawYAML, &dataFromYAML); err != nil {
		return fmt.Errorf("Unmarshaling YAML failed: %w", err)
	}

	dataFromYAML = convert(dataFromYAML)

	var schema string
	switch format {
	case "task":
		schema = schemata.Task
	case "workflow":
		schema = schemata.Workflow
	default:
		err = errors.New("format not task or workflow")
		return fmt.Errorf("Failed to obtain schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)     // load schema
	documentLoader := gojsonschema.NewGoLoader(dataFromYAML) // load empty interface

	// fmt.Printf("RAWYAML: %v\n", dataFromYAML.value)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("Error loading data: %w", err)
	}

	if result.Valid() {
		fmt.Printf("\nSUCCESS! File is valid against %s schema\n", format)
		os.Exit(0)
	} else {
		err = errors.New("file is not valid against schema\n")
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}

// convert takes a interface{} as input and recursively converts all child
// map[interface{}]interface{} to map[string]interface{}
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
