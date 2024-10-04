/*
 * === This file is part of ALICE O² ===
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

package cfgbackend

// MockSource is a minimal mock implementation of the Source interface
type MockSource struct{}

func NewMockSource() (*MockSource, error) {
	return &MockSource{}, nil
}

func (m *MockSource) Get(key string) (string, error) {
	return "", nil
}

func (m *MockSource) GetKeysByPrefix(prefix string) ([]string, error) {
	return []string{}, nil
}

func (m *MockSource) GetRecursive(key string) (Item, error) {
	return make(Map), nil
}

func (m *MockSource) GetRecursiveYaml(key string) ([]byte, error) {
	return []byte{}, nil
}

func (m *MockSource) Exists(key string) (bool, error) {
	return false, nil
}

func (m *MockSource) IsDir(key string) bool {
	return false
}

func (m *MockSource) Put(key, value string) error {
	return nil
}

func (m *MockSource) PutRecursive(key string, item Item) error {
	return nil
}

func (m *MockSource) PutRecursiveYaml(key string, data []byte) error {
	return nil
}
