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
	"fmt"
	"io/ioutil"
	"os"

	"github.com/AliceO2Group/Control/walnut/app"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v2"
)

type yamlData struct {
	yamlData []yamlData `yaml:"entries"`
}

// Template accepts a filename and format then validate against the schema specified (either workflow or task)
func Template(filename string, format string) {

	rawYAML, err := ioutil.ReadFile(filename) // import YAML file
	if err != nil {
		fmt.Println("ReadFile failed.")
		os.Exit(1)
	}

	var dataFromYAML interface{}                                           // create empty struct of expected YAML file
	if err := yaml.Unmarshal([]byte(rawYAML), &dataFromYAML); err != nil { // unmarshal YAML data into struct
		fmt.Println("Unmarshaling YAML failed.")
		os.Exit(1)
	}

	dataFromYAML = convert(dataFromYAML)

	var schema string
	switch format {
	case "task":
		schema = app.TaskSchema
	case "workflow":
		schema = app.WorkflowSchema
	default:
		fmt.Print("ERROR: Invalid format.")
		os.Exit(1)
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)     // load schema
	documentLoader := gojsonschema.NewGoLoader(dataFromYAML) // load Go struct

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		fmt.Printf("\nSUCCESS! File: %s is valid against %s schema.", filename, format)
		os.Exit(0)
	} else {
		err := "Schema validation failed."
		fmt.Printf("ERROR: %v", err)
		os.Exit(1)
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
