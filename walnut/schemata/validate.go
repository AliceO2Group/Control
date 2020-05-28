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

package schemata

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/xeipuuv/gojsonschema"
)

// Validate accepts YAML file and format then validate against the schema specified (either workflow or task)
func Validate(input []byte, format string) (err error) {

	//inputData := inputYAML{}

	var inputData interface{}
	if err := yaml.Unmarshal(input, &inputData); err != nil {
		return fmt.Errorf("Unmarshaling YAML failed: %w", err)
	}

	// inputData = convert(inputData)

	var schema string
	switch format {
	case "task":
		schema = Task
	case "workflow":
		schema = Workflow
	default:
		err = errors.New("format not task or workflow")
		return fmt.Errorf("Failed to obtain schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)  // load schema
	documentLoader := gojsonschema.NewGoLoader(inputData) // load empty interface

	// fmt.Printf("RAWYAML: %v\n", inputData.value)

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

/*
type inputYAML map[string]interface{}

func (ms *inputYAML) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var result map[interface{}]interface{}
	err := unmarshal(&result)
	if err != nil {
		panic(err)
	}
	*ms = cleanUpInterfaceMap(result)
	return nil
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) inputYAML {
	result := make(inputYAML)
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

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
*/
