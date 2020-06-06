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

package converter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AliceO2Group/Control/core/workflow"
)

func importer(filename string) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var inputJSON string
	// TODO: Probably a better way to handle this
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] == '{' {
			inputJSON += line + "\n"
			for scanner.Scan() {
				inputJSON += scanner.Text() + "\n"
			}
			break
		}
	}
	// fmt.Printf("JSON: %s", json)

	var data workflow.Role
	if err := json.Unmarshal([]byte(inputJSON), &data); err != nil {
		return fmt.Errorf("Unmarshaling Error: %w", err)
	}

	fmt.Printf("workflow.Role: %s", data)

	// var DPL workflow.Role

	return nil
}
