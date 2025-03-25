/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2024 CERN and copyright holders of ALICE O².
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

package dcs

import (
	"testing"

	dcspb "github.com/AliceO2Group/Control/core/integration/dcs/protos"
)

func TestDCSDetectorOpAvailabilityMap_compatibleWithDCSOperation(t *testing.T) {
	type args struct {
		conditionState dcspb.DetectorState
	}
	tests := []struct {
		name    string
		dsm     DCSDetectorOpAvailabilityMap
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "condition=PFR_AVAILABLE/states=[]",
			dsm:     map[dcspb.Detector]dcspb.DetectorState{},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    true,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=NULL",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_FT0: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_MID: dcspb.DetectorState_NULL_STATE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    true,
			wantErr: true,
		},
		{
			name: "condition=SOR_AVAILABLE/states=NULL",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_FT0: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_MID: dcspb.DetectorState_NULL_STATE,
			},
			args:    args{conditionState: dcspb.DetectorState_SOR_AVAILABLE},
			want:    true,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=NULL,PFR_UNAVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_FT0: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_MID: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_UNAVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=NULL,PFR_AVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_FT0: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_MID: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    true,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=NULL,PFR_AVAILABLE,PFR_UNAVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_FT0: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_MID: dcspb.DetectorState_NULL_STATE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_UNAVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=PFR_AVAILABLE,PFR_UNAVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_FT0: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_MID: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_UNAVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
		{
			name: "condition=SOR_AVAILABLE/states=PFR_AVAILABLE,PFR_UNAVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_FT0: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_MID: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_UNAVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_SOR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=PFR_AVAILABLE,PFR_UNAVAILABLE,DEAD,ERROR,RUN_INHIBIT",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_DEAD,
				dcspb.Detector_FT0: dcspb.DetectorState_ERROR,
				dcspb.Detector_MID: dcspb.DetectorState_RUN_INHIBIT,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_UNAVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
		{
			name: "condition=PFR_AVAILABLE/states=PFR_AVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_FT0: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_MID: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_AVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_PFR_AVAILABLE},
			want:    true,
			wantErr: false,
		},
		{
			name: "condition=SOR_AVAILABLE/states=PFR_AVAILABLE",
			dsm: map[dcspb.Detector]dcspb.DetectorState{
				dcspb.Detector_ZDC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_FT0: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_MID: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_EMC: dcspb.DetectorState_PFR_AVAILABLE,
				dcspb.Detector_HMP: dcspb.DetectorState_PFR_AVAILABLE,
			},
			args:    args{conditionState: dcspb.DetectorState_SOR_AVAILABLE},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, _, err := tt.dsm.compatibleWithDCSOperation(tt.args.conditionState)
			if (err != nil) != tt.wantErr {
				t.Errorf("compatibleWithDCSOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if (err != nil) == tt.wantErr {
				t.Logf("compatibleWithDCSOperation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got != tt.want {
				if err != nil {
					t.Errorf("compatibleWithDCSOperation() got = %v, want %v, error = %s", got, tt.want, err.Error())
				} else {
					t.Errorf("compatibleWithDCSOperation() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
