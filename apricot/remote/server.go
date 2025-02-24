/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020-2021 CERN and copyright holders of ALICE O².
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

// A Processor and ReposItory for COnfiguration Templates
package remote

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var log = logger.New(logrus.StandardLogger(), "apricot")

var (
	E_OK                                = status.New(codes.OK, "")
	E_CONFIGURATION_BACKEND_UNAVAILABLE = status.Errorf(codes.Internal, "configuration backend unavailable")
	E_BAD_INPUT                         = status.Errorf(codes.InvalidArgument, "bad request received")
)

type RpcServer struct {
	service configuration.Service
}

func NewServer(service configuration.Service) *grpc.Server {
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	apricotpb.RegisterApricotServer(s, &RpcServer{
		service: service,
	})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s
}

func (m *RpcServer) NewRunNumber(_ context.Context, _ *apricotpb.Empty) (*apricotpb.RunNumberResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	rn, err := m.service.NewRunNumber()
	return &apricotpb.RunNumberResponse{RunNumber: rn}, err
}

func (m *RpcServer) GetDefaults(_ context.Context, _ *apricotpb.Empty) (*apricotpb.StringMap, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	varStack := m.service.GetDefaults()
	return &apricotpb.StringMap{StringMap: varStack}, E_OK.Err()
}

func (m *RpcServer) GetVars(_ context.Context, _ *apricotpb.Empty) (*apricotpb.StringMap, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	varStack := m.service.GetVars()
	return &apricotpb.StringMap{StringMap: varStack}, E_OK.Err()
}

func (m *RpcServer) GetComponentConfiguration(_ context.Context, request *apricotpb.ComponentRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	var query *componentcfg.Query
	if rawPath := request.GetPath(); len(rawPath) > 0 {
		var err error
		query, err = componentcfg.NewQuery(rawPath)
		if err != nil {
			return nil, E_BAD_INPUT
		}
	} else if reqQuery := request.GetQuery(); reqQuery != nil {
		query = &componentcfg.Query{
			Component: reqQuery.Component,
			RunType:   reqQuery.RunType,
			RoleName:  reqQuery.MachineRole,
			EntryKey:  reqQuery.Entry,
		}
	} else {
		return nil, E_BAD_INPUT
	}

	var payload string
	var err error
	if request.ProcessTemplate {
		payload, err = m.service.GetAndProcessComponentConfiguration(query, request.GetVarStack())
	} else {
		payload, err = m.service.GetComponentConfiguration(query)
	}

	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) GetComponentConfigurationWithLastIndex(_ context.Context, request *apricotpb.ComponentRequest) (*apricotpb.ComponentResponseWithLastIndex, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	var query *componentcfg.Query
	if rawPath := request.GetPath(); len(rawPath) > 0 {
		var err error
		query, err = componentcfg.NewQuery(rawPath)
		if err != nil {
			return nil, E_BAD_INPUT
		}
	} else if reqQuery := request.GetQuery(); reqQuery != nil {
		query = &componentcfg.Query{
			Component: reqQuery.Component,
			RunType:   reqQuery.RunType,
			RoleName:  reqQuery.MachineRole,
			EntryKey:  reqQuery.Entry,
		}
	} else {
		return nil, E_BAD_INPUT
	}

	var payload string
	var lastIndex uint64
	var err error
	// ProcessTemplates not supported for this response
	if request.ProcessTemplate {
		return nil, E_BAD_INPUT
	} else {
		payload, lastIndex, err = m.service.GetComponentConfigurationWithLastIndex(query)
	}

	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentResponseWithLastIndex{Payload: payload, LastIndex: lastIndex}, E_OK.Err()
}

func (m *RpcServer) ResolveComponentQuery(_ context.Context, request *apricotpb.ComponentQuery) (*apricotpb.ComponentQuery, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	query := &componentcfg.Query{
		Component: request.Component,
		RunType:   request.RunType,
		RoleName:  request.MachineRole,
		EntryKey:  request.Entry,
	}

	resolved, err := m.service.ResolveComponentQuery(query)
	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentQuery{
		Component:   resolved.Component,
		RunType:     resolved.RunType,
		MachineRole: resolved.RoleName,
		Entry:       resolved.EntryKey,
	}, E_OK.Err()
}

func (m *RpcServer) GetDetectorForHost(_ context.Context, request *apricotpb.HostRequest) (*apricotpb.DetectorResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.GetDetectorForHost(request.GetHostname())
	if err != nil {
		return nil, err
	}
	return &apricotpb.DetectorResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) GetDetectorsForHosts(_ context.Context, request *apricotpb.HostsRequest) (*apricotpb.DetectorsResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.GetDetectorsForHosts(request.GetHosts())
	if err != nil {
		return nil, err
	}
	return &apricotpb.DetectorsResponse{Detectors: payload}, E_OK.Err()
}

func (m *RpcServer) GetCRUCardsForHost(_ context.Context, request *apricotpb.HostRequest) (*apricotpb.CRUCardsResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	cards, err := m.service.GetCRUCardsForHost(request.GetHostname())
	if err != nil {
		return nil, err
	}
	cardsJson, err := json.Marshal(cards)
	if err != nil {
		return nil, err
	}

	return &apricotpb.CRUCardsResponse{Cards: string(cardsJson)}, E_OK.Err()
}

func (m *RpcServer) GetEndpointsForCRUCard(_ context.Context, request *apricotpb.CardRequest) (*apricotpb.CRUCardEndpointResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	endpoints, err := m.service.GetEndpointsForCRUCard(request.GetHostname(), request.GetCardSerial())
	if err != nil {
		return nil, err
	}
	endpointsSpaceSeparated := strings.Join(endpoints, " ")

	return &apricotpb.CRUCardEndpointResponse{Endpoints: endpointsSpaceSeparated}, E_OK.Err()
}

func (m *RpcServer) GetRuntimeEntry(_ context.Context, request *apricotpb.GetRuntimeEntryRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.GetRuntimeEntry(request.Component, request.Key)
	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) SetRuntimeEntry(_ context.Context, request *apricotpb.SetRuntimeEntryRequest) (*apricotpb.Empty, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	err := m.service.SetRuntimeEntry(request.Component, request.Key, request.Value)
	if err != nil {
		return nil, err
	}
	return &apricotpb.Empty{}, E_OK.Err()
}

func (m *RpcServer) GetRuntimeEntries(_ context.Context, request *apricotpb.GetRuntimeEntriesRequest) (*apricotpb.StringMap, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	entries, err := m.service.GetRuntimeEntries(request.Component)
	if err != nil {
		return nil, err
	}
	return &apricotpb.StringMap{StringMap: entries}, E_OK.Err()
}

func (m *RpcServer) ListRuntimeEntries(_ context.Context, request *apricotpb.ListRuntimeEntriesRequest) (*apricotpb.ComponentEntriesResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.ListRuntimeEntries(request.Component)
	if err != nil {
		return nil, err
	}
	return &apricotpb.ComponentEntriesResponse{Payload: payload}, E_OK.Err()
}

func (m *RpcServer) RawGetRecursive(_ context.Context, request *apricotpb.RawGetRecursiveRequest) (*apricotpb.ComponentResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	payload, err := m.service.RawGetRecursive(request.RawPath)
	if err != nil {
		return &apricotpb.ComponentResponse{Payload: ""}, err
	}
	return &apricotpb.ComponentResponse{Payload: payload}, nil
}

func (m *RpcServer) ListDetectors(_ context.Context, request *apricotpb.DetectorsRequest) (*apricotpb.DetectorsResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	detectors, err := m.service.ListDetectors(request.GetAll)
	if err != nil {
		return nil, err
	}
	return &apricotpb.DetectorsResponse{Detectors: detectors}, nil
}

func (m *RpcServer) GetHostInventory(_ context.Context, request *apricotpb.HostGetRequest) (*apricotpb.HostEntriesResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	entries, err := m.service.GetHostInventory(request.Detector)
	if err != nil {
		return nil, err
	}
	return &apricotpb.HostEntriesResponse{Hosts: entries}, nil
}

func (m *RpcServer) GetDetectorsInventory(_ context.Context, _ *apricotpb.Empty) (*apricotpb.DetectorEntriesResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	entries, err := m.service.GetDetectorsInventory()
	if err != nil {
		return nil, err
	}
	return &apricotpb.DetectorEntriesResponse{DetectorEntries: DetectorInventoryToPbDetectorInventory(entries)}, nil
}

func (m *RpcServer) ListComponents(_ context.Context, _ *apricotpb.Empty) (*apricotpb.ComponentEntriesResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	entries, err := m.service.ListComponents()
	if err != nil {
		return nil, err
	}
	response := &apricotpb.ComponentEntriesResponse{Payload: entries}
	return response, nil
}

func (m *RpcServer) ListComponentEntries(_ context.Context, request *apricotpb.ListComponentEntriesRequest) (*apricotpb.ComponentEntriesResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	var query *componentcfg.EntriesQuery
	if rawPath := request.GetPath(); len(rawPath) > 0 {
		var err error
		query, err = componentcfg.NewEntriesQuery(rawPath)
		if err != nil {
			return nil, E_BAD_INPUT
		}
	} else if reqQuery := request.GetQuery(); reqQuery != nil {
		query = &componentcfg.EntriesQuery{
			Component: reqQuery.Component,
			RunType:   reqQuery.RunType,
			RoleName:  reqQuery.MachineRole,
		}
	} else {
		return nil, E_BAD_INPUT
	}

	entries, err := m.service.ListComponentEntries(query)
	if err != nil {
		return nil, err
	}
	response := &apricotpb.ComponentEntriesResponse{Payload: entries}
	return response, nil
}

func (m *RpcServer) ImportComponentConfiguration(_ context.Context, request *apricotpb.ImportComponentConfigurationRequest) (*apricotpb.ImportComponentConfigurationResponse, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()

	if request == nil {
		return nil, E_BAD_INPUT
	}

	pushQuery := &componentcfg.Query{
		Component: request.Query.Component,
		RunType:   request.Query.RunType,
		RoleName:  request.Query.MachineRole,
		EntryKey:  request.Query.Entry,
	}

	existingComponentUpdated, existingEntryUpdated, err := m.service.ImportComponentConfiguration(pushQuery, request.Payload, request.NewComponent)
	if err != nil {
		return nil, err
	}
	response := &apricotpb.ImportComponentConfigurationResponse{
		ExistingComponentUpdated: existingComponentUpdated,
		ExistingEntryUpdated:     existingEntryUpdated,
	}
	return response, nil
}

func (m *RpcServer) InvalidateComponentTemplateCache(_ context.Context, _ *apricotpb.Empty) (*apricotpb.Empty, error) {
	if m == nil || m.service == nil {
		return nil, E_CONFIGURATION_BACKEND_UNAVAILABLE
	}
	m.logMethod()
	m.service.InvalidateComponentTemplateCache()
	return &apricotpb.Empty{}, nil
}

func (m *RpcServer) logMethod() {
	if !viper.GetBool("verbose") {
		return
	}
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}
	fun := runtime.FuncForPC(pc)
	if fun == nil {
		return
	}
	log.WithPrefix("apricot").
		WithField("method", fun.Name()).
		Trace("handling RPC request")
}
