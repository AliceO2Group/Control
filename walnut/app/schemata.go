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

package app

// TaskSchema is the task template schema, i.e. the expected format of
// the task templates necessary for AliECS.
var TaskSchema string = `
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://example.com/product.schema.json",
    "title": "AliECS Task Descriptor",
    "description": "Validate task descriptor files for AliECS",
    "type": "object",
    "properties": {
        "name": {
            "description": "The unique identifier for a task",
            "type": "string"
        },
        "control": {
            "type": "object",
            "title": "Type of control",
            "properties": {
                "mode": {
                    "type": "string",
                    "title": "Mode",
                    "enum": [
                        "fairmq",
                        "basic",
                        "direct"
                    ]
                }
            }
        },
        "wants": {
            "description": "Amount of CPU and memory required for task",
            "type": "object",
            "properties": {
                "cpu": {
                    "description": "Amount of CPU",
                    "type": "number"
                },
                "memory": {
                    "description": "Amount of Memory",
                    "type": "number"
                }
            }
        },
        "bind": {
            "type": "array",
            "title": "Bind array schema",
            "items": {
                "type": "object",
                "required": [
                    "type"
                ],
                "properties": {
                    "name": {
                        "type": "string"
                    },
                    "type": {
                        "type": "string",
                        "enum": [
                            "pub",
                            "sub",
                            "push",
                            "pull"
                        ]
                    }
                }
            }
        },
        "command": {
            "type": "object",
            "title": "Commands for Task",
            "properties": {
                "shell": {
                    "type": "boolean"
                },
                "value": {
                    "type": "string"
                },
                "arguments": {
                    "type": "array",
                    "title": "Arguments",
                    "items": {
                        "type": "string"
                    }
                }
            }
        }
    }
}
 `

// WorkflowSchema is the workflow template schema, i.e. the expected format of
// the workflow templates necessary for AliECS.
var WorkflowSchema string = `
 {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "http://example.com/product.schema.json",
    "title": "AliECS Workflow Template",
    "description": "Workflow template schema for AliECS",
    "type": "object",
    "properties": {
        "name": {
            "description": "The name of the root role, which happens to be an aggregator role.",
            "type": "string"
        },
        "defaults": {
            "type": "object",
            "title": "Variable definitions",
            "description": "Variable definitions: defaults, overridden by anything",
            "properties": {
                "nProcessors": {
                    "type": "number",
                    "title": "Number of processors"
                },
                "nSinks": {
                    "type": "number",
                    "title": "Number of sinks"
                },
                "monitoring_qc_url": {
                    "type": "string"
                },
                "monitoring_dd_url": {
                    "type": "string"
                },
                "monitoring_readout_url": {
                    "type": "string"
                }
            }
        },
        "vars": {
            "type": "object",
            "title": "User variable definitions",
            "description": "Variable definitions: vars, override defaults, overridden by user vars."
        },
        "roles": {
            "title": "Roles",
            "description": "A list of child roles.",
            "type": "array",
            "properties": {
                "name": {
                    "description": "Parametrized name of an iterator role",
                    "type": "string"
                },
                "for": {
                    "description": "Amount of Memory",
                    "type": "object",
                    "properties": {
                        "begin": {
                            "title": "Begin",
                            "type": "number"
                        },
                        "end": {
                            "title": "End",
                            "type": "number"
                        },
                        "var": {
                            "title": "variable",
                            "type": "string",
                            "enum": [
                                "it"
                            ]
                        }
                    }
                },
                "connect": {
                    "title": "Connect",
                    "type": "object",
                    "properties": {
                        "name": {
                            "description": "Name of the inbound channel",
                            "type": "string"
                        },
                        "target": {
                            "description": "The target entry is a string, with some tree walk functions available for traversing the control tree.",
                            "type": "string"
                        },
                        "type": {
                            "type": "string",
                            "enum": [
                                "pub",
                                "sub",
                                "push",
                                "pull"
                            ]
                        },
                        "sndBufSize": {
                            "title": "Send Buffer Size",
                            "type": "number"
                        },
                        "rcvBufSize": {
                            "title": "Receive Buffer Size",
                            "type": "number"
                        },
                        "rateLogging": {
                            "title": "Rate Logging",
                            "type": "number"
                        }
                    }
                },
                "task": {
                    "title": "Task entry",
                    "type": "object",
                    "properties": {
                        "load": {
                            "title": "Task template to load",
                            "type": "string"
                        }
                    }
                }
            }
        }
    }
}
 `

// DPLSchema is the DPL Dump schema, the format that DPL dumps
// generated by O2 adhere to.
var DPLSchema string = `
 // TODO: Formulate DPL Schema
 `
