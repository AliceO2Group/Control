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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type io struct {
	Binding     string `json:"binding"`
	Origin      string `json:"origin"`
	Description string `json:"description"`
	Subspec     int    `json:"subspec"`
	Lifetime    int    `json:"lifetime"`
}

type options struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue"`
	Help         string `json:"help"`
}
type workflowEntry struct {
	Name               string    `json:"name"`
	Inputs             []io      `json:"inputs"`
	Ouputs             []io      `json:"outputs"`
	Options            []options `json:"options"`
	Rank               int       `json:"rank"`
	NSlots             int       `json:"nSlots"`
	InputTimeSliceID   int       `json:"inputTimeSliceId"`
	MaxInputTimeslices int       `json:"maxInputTimeslices"`
}

type metadataEntry struct {
	Name            string    `json:"name"`
	Executable      string    `json:"executable"`
	CmdlLineArgs    []string  `json:"cmdLineArgs"`
	WorkflowOptions []options `json:"workflowOptions"`
}

// Dump is a 1:1 struct representation of a DPL Dump
type Dump struct {
	Workflows []workflowEntry `json:"workflow"`
	Metadata  []metadataEntry `json:"metadata"`
}

func JSONImporter(input *os.File) (importedJSON Dump, err error) {
	byteValue, err := ioutil.ReadAll(input)
	if err != nil {
		return importedJSON, fmt.Errorf("reading file failed: %w", err)
	}

	err = json.Unmarshal(byteValue, &importedJSON)
	if err != nil {
		return importedJSON, fmt.Errorf("JSON Unmarshal failed: %w", err)
	}

	return importedJSON, nil
}
