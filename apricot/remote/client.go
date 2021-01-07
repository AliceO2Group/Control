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

const CALL_TIMEOUT = 30*time.Second

type RemoteService struct {
	remote rpcClient
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
		remote:      *rpcClient,
	}, nil
}

func (c *RemoteService) NewRunNumber() (runNumber uint32, err error) {
	var response *apricotpb.RunNumberResponse
	response, err = c.remote.NewRunNumber(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return 0, err
	}
	return response.GetRunNumber(), nil
}

func (c *RemoteService) GetDefaults() map[string]string {
	response, err := c.remote.GetDefaults(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
	if err != nil {
		return nil
	}
	return response.GetStringMap()
}

func (c *RemoteService) GetVars() map[string]string {
	response, err := c.remote.GetVars(context.Background(), &apricotpb.Empty{}, grpc.EmptyCallOption{})
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
		RunType:     query.Flavor,
		MachineRole: query.Rolename,
		Entry:       query.EntryKey,
		Timestamp:   query.Timestamp,
	}
	request := &apricotpb.ComponentRequest{
		QueryPath: &apricotpb.ComponentRequest_Query{Query: componentQuery},
		ProcessTemplate: processTemplate,
		VarStack:        varStack,
	}
	response, err = c.remote.GetComponentConfiguration(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) RawGetRecursive(path string) (payload string, err error) {
	var response *apricotpb.ComponentResponse
	request := &apricotpb.RawGetRecursiveRequest{RawPath: path}
	response, err = c.remote.RawGetRecursive(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return "", err
	}
	return response.GetPayload(), nil
}

func (c *RemoteService) GetRuntimeEntry(component string, key string) (payload string, err error) {
	var response *apricotpb.ComponentResponse
	request := &apricotpb.GetRuntimeEntryRequest{
		Component: component,
		Key:       key,
	}
	response, err = c.remote.GetRuntimeEntry(context.Background(), request, grpc.EmptyCallOption{})
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
	_, err = c.remote.SetRuntimeEntry(context.Background(), request, grpc.EmptyCallOption{})
	if err != nil {
		return err
	}
	return nil
}

type rpcClient struct {
	apricotpb.ApricotClient
	conn *grpc.ClientConn
}

func newRpcClient(cxt context.Context, cancel context.CancelFunc, endpoint string) *rpcClient {
	conn, err := grpc.DialContext(cxt, endpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.WithField("error", err.Error()).
			WithField("endpoint", endpoint).
			Errorf("cannot dial RPC endpoint")
		cancel()
		return nil
	}

	client := &rpcClient{
		ApricotClient: apricotpb.NewApricotClient(conn),
		conn: conn,
	}

	return client
}

