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
	"context"

	"google.golang.org/grpc"
)

type (
	EnvIDKey   struct{}
	RunTypeKey struct{}
)

func AddEnvAndRunType(ctx context.Context, envId, runType string) context.Context {
	ctx = context.WithValue(ctx, EnvIDKey{}, envId)
	ctx = context.WithValue(ctx, RunTypeKey{}, runType)
	return ctx
}

type measuredClientStream struct {
	grpc.ClientStream
	ctx        context.Context
	method     string
	metricName string
}

func (t *measuredClientStream) RecvMsg(m interface{}) error {
	metric := NewMetric(t.metricName)
	metric.AddTag("method", t.method)
	if env, ok := t.ctx.Value(EnvIDKey{}).(string); ok {
		metric.AddTag("envId", env)
	}
	if rt, ok := t.ctx.Value(RunTypeKey{}).(string); ok {
		metric.AddTag("runtype", rt)
	}
	defer TimerSendSingle(&metric, Millisecond)()

	err := t.ClientStream.RecvMsg(m)
	return err
}

type NameConvertType func(string) string

func SetupStreamClientInterceptor(metricName string, convert NameConvertType) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return &measuredClientStream{
			ClientStream: clientStream,
			ctx:          ctx,
			method:       convert(method),
			metricName:   metricName,
		}, nil
	}
}

func SetupUnaryClientInterceptor(name string, convert NameConvertType) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		metric := NewMetric(name)
		metric.AddTag("method", convert(method))
		if env, ok := ctx.Value(EnvIDKey{}).(string); ok {
			metric.AddTag("envId", env)
		}
		if rt, ok := ctx.Value(RunTypeKey{}).(string); ok {
			metric.AddTag("runtype", rt)
		}
		defer TimerSendSingle(&metric, Millisecond)()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
