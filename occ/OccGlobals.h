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

/** @file OccGlobals.h
 * @brief Global constants for OCC library.
 */

#ifndef OCC_OCCGLOBALS_H
#define OCC_OCCGLOBALS_H

#define OCC_DEFAULT_PORT 47100                  /// Fallback value for the control port
#define OCC_CONTROL_PORT_ARG "control-port"     /// Name of the boost::program_option to use for the control port parameter
#define OCC_CONTROL_PORT_ENV "OCC_CONTROL_PORT" /// Name of the env variable to query for the control port

#endif //OCC_OCCGLOBALS_H
