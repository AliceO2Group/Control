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

// Package remote implements a remote configuration backend for the configuration
// service, accessing configuration handled by a different application via gRPC.
package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"google.golang.org/grpc"
)

const CALL_TIMEOUT = 10 * time.Second

type RemoteService struct {
	cli rpcClient
}

func NewService(configUri string) (svc configuration.Service, err error) {
	var parsedUri *url.URL
	parsedUri, err = url.Parse(configUri)
	if err != nil {
		return nil, err
	}
	if parsedUri.Scheme != "apricot" {
		return nil, errors.New("remote configuration URI scheme must be apricot://")
	}
	endpoint := parsedUri.Host

	cxt, cancel := context.WithTimeout(context.Background(), CALL_TIMEOUT)

	rpcClient := newRpcClient(cxt, cancel, endpoint)
	if rpcClient == nil {
		return nil, fmt.Errorf("cannot dial apricot service at %s", endpoint)
	}
	return &RemoteService{
		cli: *rpcClient,
	}, nil
}

func (c *RemoteService) NewRunNumber() (runNumber uint32, err error) {
	var response *apricotpb.RunNumberResponse
	response, err = c.cli.NewRunNumber(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return 0, err
	}
	return response.GetRunNumber(), nil
}

func (c *RemoteService) GetDefaults() map[string]string {
	response, err := c.cli.GetDefaults(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return nil
	}
	return response.GetStringMap()
}

func (c *RemoteService) GetVars() map[string]string {
	response, err := c.cli.GetVars(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return nil
	}
	return response.GetStringMap()
}

func (c *RemoteService) GetComponentConfiguration(query *componentcfg.Query) (payload string, err error) {
	return c.getComponentConfigurationInternal(query, false, nil)
}

func (c *RemoteService) GetComponentConfigurationWithLastIndex(query *componentcfg.Query) (payload string, lastIndex uint64, err error) {
	return c.getComponentConfigurationInternalWithLastIndex(query, false, nil)
}

func (c *RemoteService) GetAndProcessComponentConfiguration(query *componentcfg.Query, varStack map[string]string) (payload string, err error) {
	return c.getComponentConfigurationInternal(query, true, varStack)
}

func (c *RemoteService) getComponentConfigurationInternal(query *componentcfg.Query, processTemplate bool, varStack map[string]string) (payload string, err error) {
	var response *apricotpb.ComponentResponse
	componentQuery := &apricotpb.ComponentQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
		Entry:       query.EntryKey,
	}
	request := &apricotpb.ComponentRequest{
		QueryPath:       &apricotpb.ComponentRequest_Query{Query: componentQuery},
		ProcessTemplate: processTemplate,
		VarStack:        varStack,
	}
	response, err = c.cli.GetComponentConfiguration(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) getComponentConfigurationInternalWithLastIndex(query *componentcfg.Query, processTemplate bool, varStack map[string]string) (payload string, lastIndex uint64, err error) {
	var response *apricotpb.ComponentResponseWithLastIndex
	componentQuery := &apricotpb.ComponentQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
		Entry:       query.EntryKey,
	}
	request := &apricotpb.ComponentRequest{
		QueryPath:       &apricotpb.ComponentRequest_Query{Query: componentQuery},
		ProcessTemplate: processTemplate,
		VarStack:        varStack,
	}
	response, err = c.cli.GetComponentConfigurationWithLastIndex(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", 0, err
	}
	return response.GetPayload(), response.GetLastIndex(), nil
}

func (c *RemoteService) ResolveComponentQuery(query *componentcfg.Query) (resolved *componentcfg.Query, err error) {
	var response *apricotpb.ComponentQuery
	componentQuery := &apricotpb.ComponentQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
		Entry:       query.EntryKey,
	}
	response, err = c.cli.ResolveComponentQuery(context.Background(), componentQuery, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	resolved = &componentcfg.Query{
		Component: response.Component,
		RunType:   response.RunType,
		RoleName:  response.MachineRole,
		EntryKey:  response.Entry,
	}
	return resolved, nil
}

func (c *RemoteService) RawGetRecursive(path string) (payload string, err error) {
	var response *apricotpb.ComponentResponse
	request := &apricotpb.RawGetRecursiveRequest{RawPath: path}
	response, err = c.cli.RawGetRecursive(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) GetDetectorForHost(hostname string) (payload string, err error) {
	var response *apricotpb.DetectorResponse
	request := &apricotpb.HostRequest{
		Hostname: hostname,
	}
	response, err = c.cli.GetDetectorForHost(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil

}

func (c *RemoteService) GetDetectorsForHosts(hosts []string) (payload []string, err error) {
	var response *apricotpb.DetectorsResponse
	request := &apricotpb.HostsRequest{
		Hosts: hosts,
	}
	response, err = c.cli.GetDetectorsForHosts(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return []string{}, err
	}
	return response.GetDetectors(), nil

}

func (c *RemoteService) GetCRUCardsForHost(hostname string) (cards []string, err error) {
	var response *apricotpb.CRUCardsResponse
	request := &apricotpb.HostRequest{
		Hostname: hostname,
	}
	response, err = c.cli.GetCRUCardsForHost(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	cardsStr := response.GetCards()
	err = json.Unmarshal([]byte(cardsStr), &cards)
	if err != nil {
		return nil, err
	}

	return cards, nil
}

func (c *RemoteService) GetEndpointsForCRUCard(hostname, cardSerial string) (endpoints []string, err error) {
	var response *apricotpb.CRUCardEndpointResponse
	request := &apricotpb.CardRequest{
		Hostname:   hostname,
		CardSerial: cardSerial,
	}
	response, err = c.cli.GetEndpointsForCRUCard(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	endpointsStr := response.GetEndpoints()
	endpoints = strings.Split(endpointsStr, " ")
	return endpoints, nil
}

func (c *RemoteService) GetLinkIDsForCRUEndpoint(hostname, cardSerial, endpoint string, onlyEnabled bool) ([]string, error) {
	request := &apricotpb.LinkIDsRequest{
		Hostname:   hostname,
		CardSerial: cardSerial,
		Endpoint:   endpoint,
	}
	response, err := c.cli.GetLinkIDsForCRUEndpoint(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetLinkIDs(), nil
}

func (c *RemoteService) GetAliasedLinkIDsForDetector(detector string, onlyEnabled bool) ([]string, error) {
	request := &apricotpb.AliasedLinkIDsRequest{
		Detector:    detector,
		OnlyEnabled: onlyEnabled,
	}
	response, err := c.cli.GetAliasedLinkIDsForDetector(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetAliasedLinkIDs(), nil
}

func (c *RemoteService) GetRuntimeEntry(component string, key string) (payload string, err error) {
	var response *apricotpb.ComponentResponse
	request := &apricotpb.GetRuntimeEntryRequest{
		Component: component,
		Key:       key,
	}
	response, err = c.cli.GetRuntimeEntry(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) SetRuntimeEntry(component string, key string, value string) (err error) {
	request := &apricotpb.SetRuntimeEntryRequest{
		Component: component,
		Key:       key,
		Value:     value,
	}
	_, err = c.cli.SetRuntimeEntry(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return err
	}
	return nil
}

func (c *RemoteService) GetRuntimeEntries(component string) (payload map[string]string, err error) {
	request := &apricotpb.GetRuntimeEntriesRequest{
		Component: component,
	}
	var response *apricotpb.StringMap
	response, err = c.cli.GetRuntimeEntries(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetStringMap(), nil
}

func (c *RemoteService) ListRuntimeEntries(component string) (payload []string, err error) {
	request := &apricotpb.ListRuntimeEntriesRequest{
		Component: component,
	}
	var response *apricotpb.ComponentEntriesResponse
	response, err = c.cli.ListRuntimeEntries(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) ListDetectors(getAll bool) (detectors []string, err error) {
	var response *apricotpb.DetectorsResponse
	response, err = c.cli.ListDetectors(context.Background(), &apricotpb.DetectorsRequest{GetAll: getAll}, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetDetectors(), nil
}

func (c *RemoteService) GetHostInventory(detector string) (hosts []string, err error) {
	var response *apricotpb.HostEntriesResponse
	request := &apricotpb.HostGetRequest{Detector: detector}
	response, err = c.cli.GetHostInventory(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetHosts(), nil
}

func (c *RemoteService) GetDetectorsInventory() (inventory map[string][]string, err error) {
	var response *apricotpb.DetectorEntriesResponse
	response, err = c.cli.GetDetectorsInventory(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	inventory = PbDetectorInventoryToDetectorInventory(response.GetDetectorEntries())
	return
}

func (c *RemoteService) ListComponents() (components []string, err error) {
	var response *apricotpb.ComponentEntriesResponse
	response, err = c.cli.ListComponents(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	components = response.GetPayload()
	return
}

func (c *RemoteService) ListComponentEntries(query *componentcfg.EntriesQuery) (entries []string, err error) {
	var response *apricotpb.ComponentEntriesResponse
	entriesQuery := &apricotpb.ComponentEntriesQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
	}
	request := &apricotpb.ListComponentEntriesRequest{
		QueryPath: &apricotpb.ListComponentEntriesRequest_Query{Query: entriesQuery},
	}

	response, err = c.cli.ListComponentEntries(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	entries = response.GetPayload()
	return
}

func (c *RemoteService) ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool) (existingComponentUpdated bool, existingEntryUpdated bool, err error) {
	var response *apricotpb.ImportComponentConfigurationResponse
	request := &apricotpb.ImportComponentConfigurationRequest{
		Query: &apricotpb.ComponentQuery{
			Component:   query.Component,
			RunType:     query.RunType,
			MachineRole: query.RoleName,
			Entry:       query.EntryKey,
		},
		Payload:      payload,
		NewComponent: newComponent,
	}

	response, err = c.cli.ImportComponentConfiguration(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	existingComponentUpdated = response.ExistingComponentUpdated
	existingEntryUpdated = response.ExistingEntryUpdated
	return
}

func (c *RemoteService) InvalidateComponentTemplateCache() {
	_, _ = c.cli.InvalidateComponentTemplateCache(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
}
