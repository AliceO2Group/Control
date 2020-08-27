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
	"path/filepath"
	"testing"

	"github.com/k0kubun/pp"

	"github.com/AliceO2Group/Control/core/workflow"
)

func TestExtractClass(t *testing.T) {
	t.Run("Testing class extraction", func(t *testing.T) {
		extractedClasses, err := ExtractTaskClasses(TestDump, "", []string{})
		pp.Print(extractedClasses)
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}
	})
}

func TestGenerateTaskTemplate(t *testing.T) {
	t.Run("Testing Task -> YAML", func(t *testing.T) {
		allTasks, err := ExtractTaskClasses(TestDump, "testPrefix", []string{"QualityControl"})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		err = GenerateTaskTemplate(allTasks, "dump", defaults)
		if err != nil {
			t.Errorf("failed to write YAML to file: %v", err)
		}
	})
}

func TestTaskToRole(t *testing.T) {
	t.Run("Testing Task -> workflow.Role", func(t *testing.T) {
		allTasks, err := ExtractTaskClasses(TestDump, "", []string{})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		role, err := workflow.LoadDPL(allTasks, "dump", extraVars)
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
		allTasks, err := ExtractTaskClasses(dump, "testPrefix", []string{})
		if err != nil {
			t.Errorf("extract Task Class failed: %v", err)
		}

		role, err := workflow.LoadDPL(allTasks, "dump", extraVars)
		if err != nil {
			t.Errorf("error loading task to role: %v", err)
		}

		err = GenerateWorkflowTemplate(role, "dump")
		if err != nil {
			t.Errorf("error converting Role to YAML: %v", err)
		}
	})
}

func TestGraft(t *testing.T) {
	t.Run("test grafting role", func(t *testing.T) {
		f1, _ := ioutil.ReadFile("small.yaml")
		root, _ := workflow.LoadWorkflow(f1)
		f2, _ := ioutil.ReadFile("dump.yaml")

		result, err := workflow.Graft(&root, "readout.host-{{ it }}", f2, "small")
		if err != nil{
			t.Error(err)
		}

		wd, _ := os.Getwd()
		err = ioutil.WriteFile(filepath.Join(wd, "test", "grafted.yaml"), result, 0644)
		if err != nil{
			t.Error(err)
		}
	})
}
