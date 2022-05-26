// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheusexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter"

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/model/pdata"
)

type prometheusExporter struct {
	name         string
	endpoint     string
	shutdownFunc func() error
	handler      http.Handler
	collector    *collector
	registry     *prometheus.Registry
}

var (
	errBlankPrometheusAddress = errors.New("expecting a non-blank address to run the Prometheus metrics handler")
	activeTimeSeries          = stats.Int64("prometheusexporter_active_time_series", "number of metrics time series currently active", stats.UnitDimensionless)
)

func metricViews() []*view.View {
	return []*view.View{
		{
			Measure:     activeTimeSeries,
			Aggregation: view.Count(),
		},
	}
}

func newPrometheusExporter(config *Config, set component.ExporterCreateSettings) (*prometheusExporter, error) {
	addr := strings.TrimSpace(config.Endpoint)
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errBlankPrometheusAddress
	}

	collector := newCollector(config, set.Logger)
	registry := prometheus.NewRegistry()
	_ = registry.Register(collector)

	return &prometheusExporter{
		name:         config.ID().String(),
		endpoint:     addr,
		collector:    collector,
		registry:     registry,
		shutdownFunc: func() error { return nil },
		handler: promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				ErrorHandling:     promhttp.ContinueOnError,
				EnableOpenMetrics: config.EnableOpenMetrics,
			},
		),
	}, nil
}

func (pe *prometheusExporter) Start(_ context.Context, _ component.Host) error {
	ln, err := net.Listen("tcp", pe.endpoint)
	if err != nil {
		return err
	}

	if err := view.Register(metricViews()...); err != nil {
		log.Fatalf(err.Error())
	}

	pe.shutdownFunc = ln.Close

	mux := http.NewServeMux()
	mux.Handle("/metrics", pe.handler)
	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()

	return nil
}

func (pe *prometheusExporter) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	n := 0
	rmetrics := md.ResourceMetrics()
	for i := 0; i < rmetrics.Len(); i++ {
		n += pe.collector.processMetrics(rmetrics.At(i))
	}
	stats.Record(context.TODO(), activeTimeSeries.M(int64(md.MetricCount())))

	return nil
}

func (pe *prometheusExporter) Shutdown(context.Context) error {
	return pe.shutdownFunc()
}
