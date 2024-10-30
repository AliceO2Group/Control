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

var defaults = map[string]string{
	"qc_config_uri": "json:///etc/flp.d/qc/readout.json",
	"user":          "test-user-1",
}

var extraVars = map[string]string{
	"readout_cfg_uri": "file:/home/flp/readout_stfb_emu.cfg",
}

var TestDump = Dump{
	Workflows: []workflowEntry{
		{
			Name:   "producer-0",
			Inputs: []io{},
			Outputs: []io{
				{
					Binding:     "out",
					Origin:      "TST",
					Description: "RAWDATA",
					Subspec:     0,
					Lifetime:    0,
				},
			},
			Options:            []options{},
			Rank:               0,
			NSlots:             1,
			InputTimeSliceID:   0,
			MaxInputTimeslices: 1,
		},
	},
	Metadata: []metadataEntry{
		{
			Name:       "internal-dpl-clock",
			Executable: "o2-qc-run-basic",
			CmdlLineArgs: []string{
				"-b",
				"--dump-workflow",
				"--dump-workflow-file",
				"dpl-dump.json",
			},
			WorkflowOptions: []options{
				{
					Name:         "config-path",
					Type:         "4",
					DefaultValue: "",
					Help:         "Absolute path to the config file. Overwrite the default paths. Do not use with no-data-sampling.",
				},
				{
					Name:         "no-data-sampling",
					Type:         "5",
					DefaultValue: "0",
					Help:         "Skips data sampling, connects directly the task to the producer.",
				},
				{
					Name:         "readers",
					Type:         "1",
					DefaultValue: "1",
					Help:         "number of parallel readers to use",
				},
				{
					Name:         "pipeline",
					Type:         "4",
					DefaultValue: "",
					Help:         "override default pipeline size",
				},
			},
			Channels: []string{
				"from_internal-dpl-clock_to_producer-0",
				"from_internal-dpl-clock_to_Dispatcher",
				"from_internal-dpl-clock_to_QC-TASK-RUNNER-QcTask",
			},
		},
		{
			Name:       "producer-0",
			Executable: "o2-qc-run-basic",
			CmdlLineArgs: []string{
				"-b",
				"--dump-workflow",
				"--dump-workflow-file",
				"dpl-dump.json",
			},
			WorkflowOptions: []options{
				{
					Name:         "config-path",
					Type:         "4",
					DefaultValue: "",
					Help:         "Absolute path to the config file. Overwrite the default paths. Do not use with no-data-sampling.",
				},
				{
					Name:         "no-data-sampling",
					Type:         "5",
					DefaultValue: "0",
					Help:         "Skips data sampling, connects directly the task to the producer.",
				},
				{
					Name:         "readers",
					Type:         "1",
					DefaultValue: "1",
					Help:         "number of parallel readers to use",
				},
				{
					Name:         "pipeline",
					Type:         "4",
					DefaultValue: "",
					Help:         "override default pipeline size",
				},
			},
			Channels: []string{
				"from_internal-dpl-clock_to_producer-0",
				"from_producer-0_to_Dispatcher",
			},
		},
	},
}

var TestJSON = `
{
    "workflow": [
        {
            "name": "producer-0",
            "inputs": [],
            "outputs": [
                {
                    "binding": "out",
                    "origin": "TST",
                    "description": "RAWDATA",
                    "subspec": 0,
                    "lifetime": 0
                }
            ],
            "options": [],
            "rank": 0,
            "nSlots": 1,
            "inputTimeSliceId": 0,
            "maxInputTimeslices": 1
        }
    ],
    "metadata": [
        {
            "name": "internal-dpl-clock",
            "executable": "o2-qc-run-basic",
            "cmdLineArgs": [
                "-b",
                "--dump-workflow",
                "--dump-workflow-file",
                "dpl-dump.json"
            ],
            "workflowOptions": [
                {
                    "name": "config-path",
                    "type": "4",
                    "defaultValue": "",
                    "help": "Absolute path to the config file. Overwrite the default paths. Do not use with no-data-sampling."
                },
                {
                    "name": "no-data-sampling",
                    "type": "5",
                    "defaultValue": "0",
                    "help": "Skips data sampling, connects directly the task to the producer."
                },
                {
                    "name": "readers",
                    "type": "1",
                    "defaultValue": "1",
                    "help": "number of parallel readers to use"
                },
                {
                    "name": "pipeline",
                    "type": "4",
                    "defaultValue": "",
                    "help": "override default pipeline size"
                }
            ],
            "channels": [
                "from_internal-dpl-clock_to_producer-0",
                "from_internal-dpl-clock_to_Dispatcher",
                "from_internal-dpl-clock_to_QC-TASK-RUNNER-QcTask"
            ]
        },
        {
            "name": "producer-0",
            "executable": "o2-qc-run-basic",
            "cmdLineArgs": [
                "-b",
                "--dump-workflow",
                "--dump-workflow-file",
                "dpl-dump.json"
            ],
            "workflowOptions": [
                {
                    "name": "config-path",
                    "type": "4",
                    "defaultValue": "",
                    "help": "Absolute path to the config file. Overwrite the default paths. Do not use with no-data-sampling."
                },
                {
                    "name": "no-data-sampling",
                    "type": "5",
                    "defaultValue": "0",
                    "help": "Skips data sampling, connects directly the task to the producer."
                },
                {
                    "name": "readers",
                    "type": "1",
                    "defaultValue": "1",
                    "help": "number of parallel readers to use"
                },
                {
                    "name": "pipeline",
                    "type": "4",
                    "defaultValue": "",
                    "help": "override default pipeline size"
                }
            ],
            "channels": [
                "from_internal-dpl-clock_to_producer-0",
                "from_producer-0_to_Dispatcher"
            ]
        }
    ]
}
`
