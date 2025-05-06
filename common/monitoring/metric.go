/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Michal Tichak <michal.tichak@cern.ch>
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

package monitoring

import (
	"fmt"
	"io"
	"time"

	lp "github.com/influxdata/line-protocol/v2/lineprotocol"
)

type (
	Tag struct {
		name  string
		value string
	}

	TagsType   []Tag
	FieldsType map[string]any
)

type Metric struct {
	name      string
	fields    FieldsType
	tags      TagsType
	timestamp time.Time
}

// Return empty metric, it is used right now mostly in string
// to report metrics correctly to influxdb use NewECSMetric
func NewDefaultMetric(name string, timestamp time.Time) Metric {
	return Metric{
		name: name, timestamp: timestamp,
	}
}

// creates empty metric with tag subsystem=ECS, which
// is used by Telegraf to send metrics from ECS to correct
// bucket
func NewMetric(name string) Metric {
	metric := NewDefaultMetric(name, time.Now())
	metric.AddTag("subsystem", "ECS")
	return metric
}

func (metric *Metric) AddTag(tagName string, value string) {
	metric.tags = append(metric.tags, Tag{name: tagName, value: value})
}

func (metric *Metric) setField(fieldName string, field any) {
	if metric.fields == nil {
		metric.fields = make(FieldsType)
	}
	metric.fields[fieldName] = field
}

func (metric *Metric) SetFieldInt64(fieldName string, field int64) {
	metric.setField(fieldName, field)
}

func (metric *Metric) SetFieldUInt64(fieldName string, field uint64) {
	metric.setField(fieldName, field)
}

func (metric *Metric) SetFieldFloat64(fieldName string, field float64) {
	metric.setField(fieldName, field)
}

func (metric *Metric) MergeFields(other *Metric) {
	for fieldName, field := range other.fields {
		if storedField, ok := metric.fields[fieldName]; ok {
			switch v := field.(type) {
			case int64:
				metric.fields[fieldName] = v + storedField.(int64)
			case uint64:
				metric.fields[fieldName] = v + storedField.(uint64)
			case float64:
				metric.fields[fieldName] = v + storedField.(float64)
			}
		} else {
			metric.fields[fieldName] = field
		}
	}
}

func Format(writer io.Writer, metrics []Metric) error {
	var enc lp.Encoder

	for _, metric := range metrics {
		enc.StartLine(metric.name)
		for _, tag := range metric.tags {
			enc.AddTag(tag.name, tag.value)
		}

		for fieldName, field := range metric.fields {
			// we cannot panic as we provide accessors only for allowed type with AddField*
			enc.AddField(fieldName, lp.MustNewValue(field))
		}
		enc.EndLine(metric.timestamp)
	}

	if err := enc.Err(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(writer, "%s", enc.Bytes())
	return err
}
