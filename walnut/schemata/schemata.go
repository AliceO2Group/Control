package schemata

const (
	Task = `
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "https://github.com/obviyus/Controlss/walnut/schemata/Task.json",
    "title": "AliECS Task Template",
    "description": "A task template gets loaded into a task.Class object and its template expressions are resolved. Such a task.Class represents a set of tasks (processes) that can be run throughout the cluster in the same way (i.e. with the same command line invocation).",
    "type": "object",
    "required": [
        "name"
    ],
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
            "description": "CPU, memory and ports required for task execution",
            "type": "object",
            "properties": {
                "cpu": {
                    "description": "Amount of CPU",
                    "type": "number"
                },
                "memory": {
                    "description": "Amount of Memory",
                    "type": "number"
                },
                "ports": {
                    "description": "Range of ports to use",
                    "type": "string"
                }
            }
        },
        "connect": {
            "title": "Connect",
            "type": "array",
            "items": {
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
                        "type": [
                            "integer",
                            "string"
                        ]
                    },
                    "rcvBufSize": {
                        "title": "Receive Buffer Size",
                        "type": [
                            "integer",
                            "string"
                        ]
                    },
                    "rateLogging": {
                        "title": "Rate Logging",
                        "type": [
                            "integer",
                            "string"
                        ]
                    }
                },
                "required": [
                    "name",
                    "type"
                ]
            }
        },
        "bind": {
            "type": "array",
            "title": "Bind array schema",
            "items": {
                "type": "object",
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
                    },
                    "sndBufSize": {
                        "title": "Send Buffer Size",
                        "type": [
                            "integer",
                            "string"
                        ]
                    },
                    "rcvBufSize": {
                        "title": "Receive Buffer Size",
                        "type": [
                            "integer",
                            "string"
                        ]
                    },
                    "rateLogging": {
                        "title": "Rate Logging",
                        "type": [
                            "integer",
                            "string"
                        ]
                    },
                    "transport": {
                        "type": "string",
                        "enum": [
                            "default",
                            "zeromq",
                            "nanomsg",
                            "shmem"
                        ]
                    }
                },
                "required": [
                    "name",
                    "type"
                ]
            }
        },
        "command": {
            "type": "object",
            "title": "Command for Task execution",
            "required": [
                "value"
            ],
            "properties": {
                "env": {
                    "type": "array",
                    "title": "Environment",
                    "items": {
                        "type": "string"
                    }
                },
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
                },
                "user": {
                    "type": "string"
                }
            }
        }
    }
}
`
	Workflow = `
{
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "https://github.com/obviyus/Control/walnut/schemata/Workflow.json",
    "title": "AliECS Workflow Template",
    "description": "Workflow template schema for AliECS",
    "definitions": {
        "roles": {
            "title": "Roles",
            "type": "object",
            "required": [
                "name"
            ],
            "properties": {
                "name": {
                    "description": "Parametrized name of an iterator role",
                    "type": "string"
                },
                "enabled": {
                    "type": "string"
                },
                "for": {
                    "description": "Amount of Memory",
                    "type": "object",
                    "properties": {
                        "begin": {
                            "title": "Begin port",
                            "type": "string"
                        },
                        "end": {
                            "title": "End port",
                            "type": "string"
                        },
                        "var": {
                            "title": "variables",
                            "type": "string"
                        },
                        "range": {
                            "title": "Port ranges",
                            "type": "string"
                        }
                    },
                    "oneOf": [
                        {
                            "required": [
                                "begin",
                                "end",
                                "var"
                            ]
                        },
                        {
                            "required": [
                                "range",
                                "var"
                            ]
                        }
                    ]
                },
                "constraints": {
                    "type": "array",
                    "properties": {
                        "attribute": {
                            "type": "string"
                        },
                        "value": {
                            "type": "string"
                        }
                    }
                },
                "vars": {
                    "type": "object",
                    "description": "User defined variables"
                },
                "connect": {
                    "title": "Connect",
                    "type": "array",
                    "items": {
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
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            },
                            "rcvBufSize": {
                                "title": "Receive Buffer Size",
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            },
                            "rateLogging": {
                                "title": "Rate Logging",
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            }
                        },
                        "required": [
                            "name",
                            "type",
                            "target"
                        ]
                    }
                },
                "bind": {
                    "type": "array",
                    "title": "Bind array schema",
                    "items": {
                        "type": "object",
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
                            },
                            "sndBufSize": {
                                "title": "Send Buffer Size",
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            },
                            "rcvBufSize": {
                                "title": "Receive Buffer Size",
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            },
                            "rateLogging": {
                                "title": "Rate Logging",
                                "type": [
                                    "integer",
                                    "string"
                                ]
                            },
                            "transport": {
                                "type": "string",
                                "enum": [
                                    "default",
                                    "zeromq",
                                    "nanomsg",
                                    "shmem"
                                ]
                            }
                        },
                        "required": [
                            "name",
                            "type"
                        ]
                    }
                },
                "task": {
                    "title": "Task entry",
                    "type": "object",
                    "required": [
                        "load"
                    ],
                    "properties": {
                        "load": {
                            "title": "Task template to load",
                            "type": "string"
                        },
                        "trigger": {
                            "type": "string"
                        },
                        "timeout": {
                            "type": "string"
                        },
                        "critical": {
                            "type": "boolean"
                        }
                    }
                },
                "roles": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/roles"
                    }
                }
            },
            "oneOf": [
                {
                    "required": [
                        "roles"
                    ]
                },
                {
                    "required": [
                        "task"
                    ]
                }
            ]
        }
    },
    "type": "object",
    "required": [
        "name"
    ],
    "properties": {
        "name": {
            "description": "The name of the root role, which happens to be an aggregator role.",
            "type": "string"
        },
        "roles": {
            "type": "array",
            "items": {
                "$ref": "#/definitions/roles"
            }
        },
        "defaults": {
            "type": "object",
            "title": "Variable definitions",
            "description": "Variable definitions: defaults, overridden by anything"
        },
        "vars": {
            "type": "object",
            "title": "User variable definitions",
            "description": "Variable definitions: vars, override defaults, overridden by user vars."
        }
    }
}
`
)
