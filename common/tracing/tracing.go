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

// Package tracing provides OpenTelemetry tracing initialisation for O² Control components.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var Tracer trace.Tracer = noop.NewTracerProvider().Tracer("")

type Span struct {
	Ctx  context.Context
	span trace.Span
}

func NewSpan(parent context.Context, name string) Span {
	ctx, span := Tracer.Start(parent, name)
	return Span{Ctx: ctx, span: span}
}

func (s *Span) End() {
	if s.span != nil {
		s.span.End()
	}
}

func (s *Span) Span() trace.Span {
	return s.span
}

// Run initialises the global TracerProvider and sets the package-level Tracer.
// It returns a shutdown function that must be called on process exit.
//
// \param ctx        parent context
// \param endpoint   OTel collector gRPC endpoint, e.g. "localhost:4317"
// \param serviceName OTLP service.name attribute, e.g. "aliecs"
func Run(ctx context.Context, endpoint string, serviceName string) (func(context.Context) error, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create OTLP exporter: %w", err)
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)

	otel.SetTracerProvider(tp)
	Tracer = otel.Tracer(serviceName)

	return tp.Shutdown, nil
}

// Stop is a convenience wrapper — call it when you already hold the shutdown func.
func Stop(ctx context.Context, shutdown func(context.Context) error) error {
	if shutdown == nil {
		return nil
	}
	return shutdown(ctx)
}
