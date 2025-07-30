package gm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

type Action string

const (
	PluginName = "gorm-metrics"

	ActionQuery  Action = "query"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionRow    Action = "row"
	ActionRaw    Action = "raw"

	gormMetricsStartKey = "gorm_metrics_start"
	gormMetricName      = "gorm_metrics_duration_seconds"

	labelAction  = "action"
	labelModel   = "model"
	labelJoins   = "joins"
	labelOutcome = "outcome"

	outcomeSuccess = "success"
	outcomeError   = "error"
)

var MetricLabels = []string{
	labelAction,
	labelModel,
	labelJoins,
	labelOutcome,
}

type GormMetrics struct {
	gorm.Plugin

	// HistogramVec is a Prometheus histogram vector to track the duration of GORM operations.
	HistogramVec *prometheus.HistogramVec
	LabelFn      func(*gorm.DB, Action) []string
}

var (
	// defaultLabelFn is the default function to generate labels for GORM metrics.
	defaultLabelFn = func(db *gorm.DB, action Action) []string {
		model := db.Statement.Table
		if model == "" {
			model = "unknown"
		}

		joins := fmt.Sprintf("%d", len(db.Statement.Joins))

		outcome := outcomeSuccess
		if db.Error != nil {
			outcome = outcomeError
		}

		return []string{
			string(action),
			strings.ToLower(model),
			joins,
			outcome,
		}
	}

	// defaultPlugin is the default GormMetrics instance with default settings.
	defaultPlugin *GormMetrics
)

// Default returns a new GormMetrics instance with default settings.
// It initializes the HistogramVec with default buckets and automatically
// registers it with Prometheus' default registry. This function is not thread-safe,
// and will panic if the metric is registration fails. It is recommended to call this
// function once at the start of your application.
// If you need to customize the metric or use different prometheus registry, create a
// new GormMetrics instance instead.
func Default() *GormMetrics {
	if defaultPlugin == nil {
		defaultPlugin = &GormMetrics{
			// n.b. promauto panics if the metric is already registered.
			HistogramVec: promauto.NewHistogramVec(prometheus.HistogramOpts{
				Name:    gormMetricName,
				Help:    "Duration of GORM operations in seconds",
				Buckets: prometheus.DefBuckets,
			}, MetricLabels),
			LabelFn: defaultLabelFn,
		}
	}
	return defaultPlugin
}

func (g *GormMetrics) Name() string {
	return PluginName
}

func (g *GormMetrics) Initialize(db *gorm.DB) error {
	// Register the metrics collector with the GORM DB instance
	if db == nil {
		return gorm.ErrInvalidDB
	}

	return anyErr(
		db.Callback().Query().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Create().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Update().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Delete().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Raw().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Row().Before("*").Register("gorm-metrics:start", g.start),
		db.Callback().Query().After("gorm:query").Register("gorm-metrics:query", func(d *gorm.DB) {
			g.observeMetrics(d, ActionQuery)
		}),
		db.Callback().Create().After("gorm:create").Register("gorm-metrics:create", func(d *gorm.DB) {
			g.observeMetrics(d, ActionCreate)
		}),
		db.Callback().Update().After("gorm:update").Register("gorm-metrics:update", func(d *gorm.DB) {
			g.observeMetrics(d, ActionUpdate)
		}),
		db.Callback().Delete().After("gorm:delete").Register("gorm-metrics:delete", func(d *gorm.DB) {
			g.observeMetrics(d, ActionDelete)
		}),
		db.Callback().Raw().After("gorm:raw").Register("gorm-metrics:raw", func(d *gorm.DB) {
			g.observeMetrics(d, ActionRaw)
		}),
		db.Callback().Row().After("gorm:row").Register("gorm-metrics:row", func(d *gorm.DB) {
			g.observeMetrics(d, ActionRow)
		}),
	)
}

func (g *GormMetrics) observeMetrics(db *gorm.DB, action Action) {
	if db.Statement.Context == nil {
		return
	}
	startTime, ok := db.Statement.Context.Value(gormMetricsStartKey).(time.Time)
	if !ok {
		return
	}

	g.HistogramVec.WithLabelValues(
		g.LabelFn(db, action)...,
	).Observe(time.Since(startTime).Seconds())
}

func (g *GormMetrics) start(db *gorm.DB) {
	db.Statement.Context = context.WithValue(db.Statement.Context, gormMetricsStartKey, time.Now())
}

func anyErr(err ...error) error {
	for _, e := range err {
		if e != nil {
			return e
		}
	}
	return nil
}
