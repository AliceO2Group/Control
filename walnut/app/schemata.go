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

// TaskSchema struct holds unmarshaled YAML data of task templates
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
