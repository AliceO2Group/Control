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

package remote

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"google.golang.org/grpc"
)

const CALL_TIMEOUT = 10*time.Second

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
		Timestamp:   query.Timestamp,
	}
	request := &apricotpb.ComponentRequest{
		QueryPath: &apricotpb.ComponentRequest_Query{Query: componentQuery},
		ProcessTemplate: processTemplate,
		VarStack:        varStack,
	}
	response, err = c.cli.GetComponentConfiguration(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
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

func (c *RemoteService) GetCRUCardsForHost(hostname string) (cars []string, err error) {
	var response *apricotpb.CRUCardsResponse
	request := &apricotpb.HostRequest{
		Hostname: hostname,
	}
	response, err = c.cli.GetCRUCardsForHost(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return nil, err
	}
	return response.GetCards(), nil
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

func (c *RemoteService) ListComponents() (components []string, err error) {
	var response *apricotpb.ComponentEntriesResponse
	response, err = c.cli.ListComponents(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	components = response.GetPayload()
	return
}

func (c *RemoteService) ListComponentEntries(query *componentcfg.EntriesQuery, showLatestTimestamp bool) (entries []string, err error) {
	var response *apricotpb.ComponentEntriesResponse
	entriesQuery := &apricotpb.ComponentEntriesQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
	}
	request := &apricotpb.ListComponentEntriesRequest{
		QueryPath:         &apricotpb.ListComponentEntriesRequest_Query{Query: entriesQuery},
		IncludeTimestamps: showLatestTimestamp,
	}

	response, err = c.cli.ListComponentEntries(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	entries = response.GetPayload()
	return
}

func (c *RemoteService) ListComponentEntryHistory(query *componentcfg.Query) (entries []string, err error) {
	var response *apricotpb.ComponentEntriesResponse
	request := &apricotpb.ComponentQuery{
		Component:   query.Component,
		RunType:     query.RunType,
		MachineRole: query.RoleName,
		Entry:       query.EntryKey,
		Timestamp:   "",
	}

	response, err = c.cli.ListComponentEntryHistory(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return
	}
	entries = response.GetPayload()
	return
}

func (c *RemoteService) ImportComponentConfiguration(query *componentcfg.Query, payload string, newComponent bool, useVersioning bool) (existingComponentUpdated bool, existingEntryUpdated bool, newTimestamp int64, err error) {
	var response *apricotpb.ImportComponentConfigurationResponse
	request := &apricotpb.ImportComponentConfigurationRequest{
		Query:         &apricotpb.ComponentQuery{
			Component:   query.Component,
			RunType:     query.RunType,
			MachineRole: query.RoleName,
			Entry:       query.EntryKey,
			Timestamp:   query.Timestamp,
		},
		Payload:       payload,
		NewComponent:  newComponent,
		UseVersioning: useVersioning,
	}


	response, err = c.cli.ImportComponentConfiguration(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return
	}

	existingComponentUpdated = response.ExistingComponentUpdated
	existingEntryUpdated = response.ExistingEntryUpdated
	newTimestamp = response.NewTimestamp
	return
}
