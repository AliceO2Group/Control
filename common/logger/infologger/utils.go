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

package infologger

import "github.com/sirupsen/logrus"

const (
	IL_Ops     = 1
	IL_Support = 6
	IL_Devel   = 11
	IL_Trace   = 21
)

// Severity/priority constants:
// https://github.com/AliceO2Group/InfoLogger/blob/master/include/InfoLogger/InfoLogger.hxx

// Extract InfoLogger level number from logrus Level
func logrusEntryToInfoLoggerLevel(e *logrus.Entry) string {
	/// Some predefined constants that may be used to set the message level
	/// (i.e. visibility of the message, based on who reads it)
	/// The level is an integer in the 1-99 range (1: topmost visibility)
	/// The enum below provides the main boundaries for typical operations,
	/// and one may use finer granularity within each range.
	/// operations (1-5) support (6-10) developer (11-20) trace (21-99).
	/// Trace messages should typically not be enabled in normal running conditions,
	/// and usually related to debugging activities (also akin to the 'Debug' severity).
	// enum Level {
	//  Ops = 1,
	//  Support = 6,
	//  Devel = 11,
	//  Trace = 21
	// };
	switch e.Level {
	case logrus.TraceLevel:
		return "21"
	case logrus.DebugLevel:
		return "11"
	default:
		return "1"
	}

}

// Extract InfoLogger severity char from logrus Level
func logrusLevelToInfoLoggerSeverity(level logrus.Level) string {
	switch level {
	case logrus.PanicLevel:
		return "F"
	case logrus.FatalLevel:
		return "F"
	case logrus.ErrorLevel:
		return "E"
	case logrus.WarnLevel:
		return "W"
	case logrus.InfoLevel:
		return "I"
	case logrus.DebugLevel:
		return "D"
	case logrus.TraceLevel:
		return "D"
	default:
		return "U"
	}
}

func buildFineFacility(baseFacility string, data logrus.Fields) string {
	if data == nil {
		return baseFacility
	}

	if prefix, ok := data["prefix"]; ok {
		return baseFacility + "/" + prefix.(string)
	}
	return baseFacility
}
