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
	"io/ioutil"
	"testing"
)

func TestTaskSchemaValidation(t *testing.T) {
	t.Run("Testing Validation with Task", func(t *testing.T) {
		testfile, _ := ioutil.ReadFile("task_test.yaml")
		err := CheckSchema(testfile, "task")
		if err != nil {
			t.Errorf("task validation failed: %v", err)
		}
	})
}

func TestWorkflowSchemaValidation(t *testing.T) {
	t.Run("Testing Validation with Workflow", func(t *testing.T) {
		testfile, _ := ioutil.ReadFile("workflow_test.yaml")
		err := CheckSchema(testfile, "workflow")
		if err != nil {
			t.Errorf("workflow validation failed: %v", err)
		}
	})
}
