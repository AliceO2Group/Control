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

// Validate takes a YAML file and format then perform schema validation
// on the file against the schema specified (either workflow or task)
func Validate(input []byte, format string) (err error) {

	var inputData interface{}
	if err := yaml.Unmarshal(input, &inputData); err != nil {
		return fmt.Errorf("Unmarshaling YAML failed: %w", err)
	}

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
	documentLoader := gojsonschema.NewGoLoader(inputData) // load unmarhsaled YAML

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("Error loading data: %w", err)
	}

	if result.Valid() {
		os.Exit(0)
	} else {
		return fmt.Errorf("%s", result.Errors()[0])
	}

	return nil
}
