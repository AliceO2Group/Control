/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

package template

import (
	"fmt"
	"io"
	"strings"

	"github.com/AliceO2Group/Control/core/the"
)

// Implements pongo2.TemplateLoader to fetch included templates from Consul paths
type ConsulTemplateLoader struct {
	basePath string
}

func NewConsulTemplateLoader(basePath string) *ConsulTemplateLoader {
	return &ConsulTemplateLoader{
		basePath: basePath,
	}
}

func (c *ConsulTemplateLoader) Abs(base, name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}
	if strings.HasPrefix(name, c.basePath) {
		return name
	}

	if c.basePath == "" {
		return name
	}
	return c.basePath + "/" + name
}

func (c *ConsulTemplateLoader) Get(path string) (io.Reader, error) {
	payload, err := the.ConfSvc().GetComponentConfiguration(path)
	if err != nil {
		log.WithError(err).
			WithField("path", path).
			Warn("failed to include component configuration in template")
		return strings.NewReader(fmt.Sprintf("{\"error\":\"%s\"}", err.Error())), err
	}
	return strings.NewReader(payload), nil
}
