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
	"io/ioutil"
	"os"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/AliceO2Group/Control/core/workflow"
)

func TestExtractClass(t *testing.T) {
	t.Run("Testing class extraction", func(t *testing.T) {
		extractedClasses, err := ExtractTaskClasses(TestDump, []string{})
		pp.Print(extractedClasses)
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}
	})
}

func TestGenerateTaskTemplate(t *testing.T) {
	t.Run("Testing Task -> YAML", func(t *testing.T) {
		allTasks, err := ExtractTaskClasses(TestDump, []string{"QualityControl"})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		err = GenerateTaskTemplate(allTasks, "dump")
		if err != nil {
			t.Errorf("failed to write YAML to file: %v", err)
		}
	})
}

func TestTaskToRole(t *testing.T) {
	t.Run("Testing Task -> workflow.Role", func(t *testing.T) {
		allTasks, err := ExtractTaskClasses(TestDump, []string{})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		role, err := workflow.LoadDPL(allTasks, "dump")
		if err != nil {
			t.Errorf("error loading task to role: %v", err)
		}
		_, _ = pp.Print(role)
	})
}

func TestGenerateWorkflowTemplate(t *testing.T) {
	t.Run("Testing Role -> workflow.yaml", func(t *testing.T) {
		dumpFile, err := ioutil.ReadFile("dump.json")
		dump, err := DPLImporter(dumpFile)
		allTasks, err := ExtractTaskClasses(dump, []string{})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		role, err := workflow.LoadDPL(allTasks, "dump")
		if err != nil {
			t.Errorf("error loading task to role: %v", err)
		}

		err = GenerateWorkflowTemplate(role, false, "dump")
		if err != nil {
			t.Errorf("error converting Role to YAML: %v", err)
		}
	})
}

func TestLoadWorkflow(t *testing.T) {
	t.Run("test loading WFT", func(t *testing.T) {
		f, err := ioutil.ReadFile("readout-qc.yaml")
		if err != nil {
			t.Errorf("opening file failed: %s", err)
		}

		i, a, err := workflow.LoadWorkflow(f)
		if err != nil {
			t.Errorf("load workflow failed: %s", err)
		}

		var holder workflow.Role
		if i == nil {
			holder = a
		} else if a == nil {
			holder = i
		}

		wd, _ := os.Getwd()
		_ = GenerateWorkflowTemplate(holder, i == nil, wd+"/test")
	})
}

func TestIteratorUnmarshal(t *testing.T) {
	t.Run("unmarshal iterator", func(t *testing.T) {
		f, _ := ioutil.ReadFile("readout-qc.yaml")
		_, _ = workflow.UnmarshalIterator(f)
	})
}
