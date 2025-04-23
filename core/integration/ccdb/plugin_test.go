/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka
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

package ccdb

import (
	"github.com/AliceO2Group/Control/common/runtype"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("NewGRPObject", func() {
	var varStack map[string]string

	BeforeEach(func() {
		varStack = map[string]string{
			"environment_id":             "2oDvieFrVTi",
			"run_number":                 "123456",
			"run_type":                   "PHYSICS",
			"run_start_time_ms":          "10000",
			"run_end_completion_time_ms": "20000",
			"trg_start_time_ms":          "10100",
			"trg_end_time_ms":            "19900",
			"detectors":                  `["ITS","TPC"]`,
			"pdp_n_hbf_per_tf":           "128",
			"lhc_period":                 "LHC22a",
			"ctp_readout_enabled":        "true",
		}
	})

	It("should create a valid GRP object with all fields set", func() {
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.runNumber).To(Equal(uint32(123456)))
		Expect(grp.lhcPeriod).To(Equal("LHC22a"))
		Expect(grp.runType).To(Equal(runtype.PHYSICS))
		Expect(grp.runStartTimeMs).To(Equal("10000"))
		Expect(grp.runEndCompletionTimeMs).To(Equal("20000"))
		Expect(grp.trgStartTimeMs).To(Equal("10100"))
		Expect(grp.trgEndTimeMs).To(Equal("19900"))
		Expect(grp.detectors).To(ContainElements("ITS", "TPC", "TRG")) // TRG added due to ctp_readout_enabled
		Expect(grp.hbfPerTf).To(Equal(uint32(128)))
	})

	It("should return nil when environment_id is missing", func() {
		delete(varStack, "environment_id")
		grp := NewGRPObject(varStack)
		Expect(grp).To(BeNil())
	})

	It("should return nil when run_number is missing", func() {
		delete(varStack, "run_number")
		grp := NewGRPObject(varStack)
		Expect(grp).To(BeNil())
	})

	It("should handle invalid run_number format", func() {
		varStack["run_number"] = "invalid"
		grp := NewGRPObject(varStack)
		Expect(grp).To(BeNil())
	})

	It("should handle missing run_type", func() {
		delete(varStack, "run_type")
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.runType).To(Equal(runtype.NONE))
	})

	It("should return nil when pdp_n_hbf_per_tf is missing", func() {
		delete(varStack, "pdp_n_hbf_per_tf")
		grp := NewGRPObject(varStack)
		Expect(grp).To(BeNil())
	})

	It("should override run start time for synthetic runs and compute run end time accordingly", func() {
		varStack["run_type"] = "SYNTHETIC"
		varStack["pdp_override_run_start_time"] = "15000"
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.runStartTimeMs).To(Equal("15000"))
		Expect(grp.runEndCompletionTimeMs).To(Equal("25000"))
	})

	It("should log a warning for run start time override in non-synthetic runs, but override it anyway", func() {
		// overriding is not really a strong requirement, it could be changed if requested
		varStack["pdp_override_run_start_time"] = "15000"
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.runStartTimeMs).To(Equal("15000"))
		Expect(grp.runEndCompletionTimeMs).To(Equal("25000"))
	})

	It("should survive empty detectors list", func() {
		varStack["detectors"] = "[]"
		varStack["ctp_readout_enabled"] = "false"
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.detectors).To(BeEmpty())
	})

	It("should return nil when there is invalid detectors JSON", func() {
		varStack["detectors"] = "invalid json"
		grp := NewGRPObject(varStack)
		Expect(grp).To(BeNil())
	})

	It("should handle missing trg_start_time_ms and trg_end_time_ms", func() {
		delete(varStack, "trg_start_time_ms")
		delete(varStack, "trg_end_time_ms")
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.trgStartTimeMs).To(BeEmpty())
		Expect(grp.trgEndTimeMs).To(BeEmpty())
	})

	It("should handle synthetic run with original run number", func() {
		varStack["run_type"] = "SYNTHETIC"
		varStack["original_run_number"] = "654321"
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.originalRunNumber).To(Equal(uint32(654321)))
		Expect(grp.runNumber).To(Equal(uint32(123456)))
	})

	It("should ignore original run number for non-synthetic runs", func() {
		varStack["original_run_number"] = "654321"
		grp := NewGRPObject(varStack)
		Expect(grp).ToNot(BeNil())
		Expect(grp.originalRunNumber).To(Equal(uint32(0)))
	})

	// fixme: we do not test extracting the list of FLPs, because we would need to mock an env manager with a realistic enough environment
	//  once it is easier to mock it, we should add a test for it
})

var _ = Describe("NewCcdbGrpWriteCommand", func() {
	var plugin *Plugin
	var grp *GeneralRunParameters

	BeforeEach(func() {
		plugin = &Plugin{
			ccdbUrl: "http://ccdb-test:8080",
		}
		grp = &GeneralRunParameters{
			runNumber:              123456,
			runType:                runtype.PHYSICS,
			detectors:              []string{"ITS", "TPC"},
			runStartTimeMs:         "10000",
			runEndCompletionTimeMs: "20000",
			trgStartTimeMs:         "10100",
			trgEndTimeMs:           "19900",
			hbfPerTf:               128,
			lhcPeriod:              "LHC22a",
		}
	})

	It("should create basic command with required fields", func() {
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).To(ContainSubstring(" -r 123456"))
		Expect(cmd).To(ContainSubstring(" -p LHC22a"))
		Expect(cmd).To(ContainSubstring(" -t 1")) // PHYSICS enum value
		Expect(cmd).To(ContainSubstring(" -n 128"))
		Expect(cmd).To(ContainSubstring(" -s 10000"))
		Expect(cmd).To(ContainSubstring(" -e 20000"))
		Expect(cmd).To(ContainSubstring(" --start-time-ctp 10100"))
		Expect(cmd).To(ContainSubstring(" --end-time-ctp 19900"))
		Expect(cmd).To(ContainSubstring(" --ccdb-server http://ccdb-test:8080"))
	})

	It("should return an error when run number is 0", func() {
		grp.runNumber = 0
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).To(HaveOccurred())
		Expect(cmd).To(BeEmpty())
	})

	It("should return an error when LHC period is missing", func() {
		grp.lhcPeriod = ""
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).To(HaveOccurred())
		Expect(cmd).To(BeEmpty())
	})

	It("should add the refresh flag when requested", func() {
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).To(ContainSubstring(" --refresh"))
	})

	It("should not add the refresh flag when not requested", func() {
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).ToNot(ContainSubstring(" --refresh"))
	})

	It("should include detector lists when present", func() {
		grp.continuousReadoutDetectors = []string{"ITS"}
		grp.triggeringDetectors = []string{"TPC"}
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).To(ContainSubstring(` -d "ITS,TPC"`))
		Expect(cmd).To(ContainSubstring(` -c "ITS"`))
		Expect(cmd).To(ContainSubstring(` -g "TPC"`))
	})

	It("should include FLP list when present", func() {
		grp.flpIdList = []string{"flp-1", "flp-2"}
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).To(ContainSubstring(` -f "flp-1,flp-2"`))
	})

	It("should include original run number for synthetic runs", func() {
		grp.originalRunNumber = 654321
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).To(ContainSubstring(" -o 654321"))
	})

	It("should skip optional fields when empty", func() {
		grp = &GeneralRunParameters{
			runNumber: 123456,
			lhcPeriod: "LHC22a",
		}
		cmd, err := plugin.NewCcdbGrpWriteCommand(grp, "http://ccdb-test:8080", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(cmd).NotTo(ContainSubstring(" -d"))
		Expect(cmd).NotTo(ContainSubstring(" -c"))
		Expect(cmd).NotTo(ContainSubstring(" -g"))
		Expect(cmd).NotTo(ContainSubstring(" -f"))
		Expect(cmd).NotTo(ContainSubstring(" -t"))
		Expect(cmd).NotTo(ContainSubstring(" -s"))
		Expect(cmd).NotTo(ContainSubstring(" -e"))
		Expect(cmd).NotTo(ContainSubstring(" --start-time-ctp"))
		Expect(cmd).NotTo(ContainSubstring(" --end-time-ctp"))
	})
})

func TestCcdbGrpPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CCDB GRP integration plugin Test Suite")
}
