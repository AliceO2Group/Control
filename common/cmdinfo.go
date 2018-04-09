/*
 * === This file is part of octl <https://github.com/teo/octl> ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

package common

type CommandInfo struct {
	Env       []string `json:"env,omitempty"`
	Shell     *bool    `json:"shell,omitempty"`
	Value     *string   `json:"value,omitempty"`
	Arguments []string `json:"arguments,omitempty"`
	User      *string  `json:"user,omitempty"`
}

func (m *CommandInfo) Reset() { *m = CommandInfo{} }

func (m *CommandInfo) Copy() *CommandInfo {
	cmd := CommandInfo{
		Env:       append([]string{}, m.Env...),
		Shell:     nil,
		Value:     nil,
		Arguments: append([]string{}, m.Arguments...),
		User:      nil,
	}
	if m.Shell != nil {
		*cmd.Shell = *m.Shell
	}
	if m.Value != nil {
		*cmd.Value = *m.Value
	}
	if m.User != nil {
		*cmd.User = *m.User
	}
	return &cmd
}

func (m *CommandInfo) Equals(other *CommandInfo) (response bool) {
	response = true
	if m == nil || other == nil {
		return false
	}

	if len(m.Env) != len(other.Env) ||
		len(m.Arguments) != len(other.Arguments) {
		return false
	}

	for i, _ := range m.Env {
		if m.Env[i] != other.Env[i] {
			return false
		}
	}
	for i, _ := range m.Arguments {
		if m.Arguments[i] != other.Arguments[i] {
			return false
		}
	}
	if !((m.Value == nil && other.Value == nil) ||
		 *m.Value == *other.Value) {
		return false
	}
	if !((m.User == nil && other.User == nil) ||
		 *m.User == *other.User) {
		return false
	}
	if !((m.Shell == nil && other.Shell == nil) ||
		 *m.Shell == *other.Shell) {
		return false
	}
	return
}

func (m *CommandInfo) UpdateFrom(n *CommandInfo) {
	// empty slice updates
	// nil slice does NOT update
	if n.Env != nil {
		m.Env = append([]string{}, n.Env...)
	}
	if n.Shell != nil {
		*m.Shell = *n.Shell
	}
	if n.Value != nil {
		*m.Value = *n.Value
	}
	if n.Arguments != nil {
		m.Arguments = append([]string{}, n.Arguments...)
	}
	if n.User != nil {
		*m.User = *n.User
	}
}

const defaultCommandInfoShell = false

func (m *CommandInfo) GetEnv () []string {
	if m != nil {
		return m.Env
	}
	return nil
}

func (m *CommandInfo) GetShell() bool {
	if m != nil && m.Shell != nil {
		return *m.Shell
	}
	return defaultCommandInfoShell
}

func (m *CommandInfo) GetValue() string {
	if m != nil {
		return *m.Value
	}
	return ""
}

func (m *CommandInfo) GetArguments() []string {
	if m != nil {
		return m.Arguments
	}
	return nil
}

func (m *CommandInfo) GetUser() string {
	if m != nil && m.User != nil {
		return *m.User
	}
	return ""
}