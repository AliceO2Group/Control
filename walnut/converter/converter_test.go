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
	"github.com/AliceO2Group/Control/core/workflow"
	"github.com/k0kubun/pp"
	"os"
	"testing"
)

func TestExtractClass(t *testing.T) {
	t.Run("Testing class extraction", func(t *testing.T) {
		file, err := os.Open("dump.json")
		if err != nil {
			t.Errorf("Opening file failed: %w", err)
		}
		defer file.Close()

		DplDump, err := JSONImporter(file)
		if err != nil {
			t.Errorf("Import failed: %w", err)
		}

		_, err = ExtractTaskClasses(DplDump)
		if err != nil {
			t.Errorf("Extract Task Class failed: %w", err)
		}
	})
}

func TestTaskToYAML(t *testing.T) {
	t.Run("Testing Task -> YAML", func(t *testing.T) {
		file, err := os.Open("dump.json")
		if err != nil {
			t.Errorf("Opening file failed: %w", err)
		}
		defer file.Close()

		DplDump, err := JSONImporter(file)
		if err != nil {
			t.Errorf("Import failed: %w", err)
		}

		allTasks, err := ExtractTaskClasses(DplDump)
		if err != nil {
			t.Errorf("Extract Task Class failed: %w", err)
		}

		err = TaskToYAML(allTasks)
		if err != nil {
			t.Errorf("Filed to write YAML to file: %w", err)
		}
	})
}

func TestTaskToRole(t *testing.T) {
	t.Run("Testing Task -> workflow.Role", func(t *testing.T) {
		file, err := os.Open("dump.json")
		if err != nil {
			t.Errorf("Opening file failed: %w", err)
		}
		defer file.Close()

		DplDump, err := JSONImporter(file)
		if err != nil {
			t.Errorf("Import failed: %w", err)
		}

		allTasks, err := ExtractTaskClasses(DplDump)
		if err != nil {
			t.Errorf("Extract Task Class failed: %w", err)
		}

		role, err := workflow.LoadDPL(allTasks)
		if err != nil {
			t.Errorf("error loading task to role: %w", err)
		}

		pp.Print(role)

	})
}
