/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.guyot@cern.ch>
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

package remote

import apricotpb "github.com/AliceO2Group/Control/apricot/protos"

func DetectorInventoryToPbDetectorInventory(inventory map[string][]string) map[string]*apricotpb.DetectorInventoryResponse {
	response := make(map[string]*apricotpb.DetectorInventoryResponse)
	var inventoryResp *apricotpb.DetectorInventoryResponse
	for det, flps := range inventory {
		inventoryResp = &apricotpb.DetectorInventoryResponse{
			Flps: flps,
		}
		response[det] = inventoryResp
	}
	return response
}

func PbDetectorInventoryToDetectorInventory(inventory map[string]*apricotpb.DetectorInventoryResponse) map[string][]string {
	response := make(map[string][]string)
	for det, flps := range inventory {
		response[det] = flps.Flps
	}
	return response
}
