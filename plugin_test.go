package gm_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	io_prometheus_client "github.com/prometheus/client_model/go"
	gm "github.com/tpokki/gorm-metrics"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Person struct {
	gorm.Model

	Name string
	Age  int
}

type FavoriteColor struct {
	PersonID uint
	Name     string
}

func TestSimple(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	plugin := gm.Default()
	if err := db.Use(plugin); err != nil {
		t.Fatalf("failed to use plugin: %v", err)
	}
	if err := db.AutoMigrate(&Person{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	person := &Person{Name: "Joe", Age: 30}
	if err := db.Create(person).Error; err != nil {
		t.Fatalf("failed to create test model: %v", err)
	}

	// Perform 10 operations to trigger metrics
	for range 10 {
		if err := db.First(&person).Error; err != nil {
			t.Fatalf("failed to query test model: %v", err)
		}
	}

	if person.Name != "Joe" || person.Age != 30 {
		t.Fatalf("expected person model to have Name 'Joe' and Age 30, got Name '%s' and Age %d", person.Name, person.Age)
	}
	if err := db.Delete(person).Error; err != nil {
		t.Fatalf("failed to delete person model: %v", err)
	}
	if err := db.First(&person).Error; err == nil {
		t.Fatalf("expected person model to be deleted, but it was found")
	}

	var value io_prometheus_client.Metric

	// verify successful query metric
	metric, err := plugin.HistogramVec.MetricVec.GetMetricWithLabelValues("default", "query", "people", "0", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 10 {
		t.Fatalf("expected sample count to be 10, got %d", value.GetHistogram().GetSampleCount())
	}

	// verify failed query metric (query against empty table)
	metric, err = plugin.HistogramVec.MetricVec.GetMetricWithLabelValues("default", "query", "people", "0", "error")
	if err != nil {
		t.Fatalf("failed to get error metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write error metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 1 {
		t.Fatalf("expected sample count to be 1, got %d", value.GetHistogram().GetSampleCount())
	}

	// verify delete metric
	metric, err = plugin.HistogramVec.MetricVec.GetMetricWithLabelValues("default", "delete", "people", "0", "success")
	if err != nil {
		t.Fatalf("failed to get delete metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write delete metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected delete histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 1 {
		t.Fatalf("expected delete sample count to be 1, got %d", value.GetHistogram().GetSampleCount())
	}
}

func TestJoins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	plugin := gm.Default()
	if err := db.Use(plugin); err != nil {
		t.Fatalf("failed to use plugin: %v", err)
	}
	if err := db.AutoMigrate(&Person{}, &FavoriteColor{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	person := &Person{Name: "Jill", Age: 30}
	if err := db.Create(person).Error; err != nil {
		t.Fatalf("failed to create person model: %v", err)
	}

	favoriteColor := &FavoriteColor{Name: "Red", PersonID: person.ID}
	if err := db.Create(favoriteColor).Error; err != nil {
		t.Fatalf("failed to create favorite color model: %v", err)
	}

	var result struct {
		Person
		FavoriteColor
	}

	if err := db.Model(&Person{}).Joins("LEFT JOIN favorite_colors ON favorite_colors.person_id = people.id").First(&result).Error; err != nil {
		t.Fatalf("failed to perform join query: %v", err)
	}

	if result.Person.Name != "Jill" || result.Age != 30 {
		t.Fatalf("expected joined model to have Name 'Jill' and Age 30, got Name '%s' and Age %d", result.Person.Name, result.Age)
	}

	var value io_prometheus_client.Metric
	// verify successful join query metric
	metric, err := plugin.HistogramVec.MetricVec.GetMetricWithLabelValues("default", "query", "people", "1", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 1 {
		t.Fatalf("expected sample count to be 1, got %d", value.GetHistogram().GetSampleCount())
	}
}

func TestNamedCtx(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	plugin := gm.Default()
	if err := db.Use(plugin); err != nil {
		t.Fatalf("failed to use plugin: %v", err)
	}
	if err := db.AutoMigrate(&Person{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	person := &Person{Name: "Alice", Age: 25}
	if err := db.WithContext(gm.WithName("test_name")).Create(person).Error; err != nil {
		t.Fatalf("failed to create person model: %v", err)
	}

	var value io_prometheus_client.Metric
	// verify successful query metric with named context
	metric, err := plugin.HistogramVec.MetricVec.GetMetricWithLabelValues("test_name", "create", "people", "0", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 1 {
		t.Fatalf("expected sample count to be 1, got %d", value.GetHistogram().GetSampleCount())
	}
}

func TestCustomHistogram(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm DB: %v", err)
	}

	// customplugin that collects metrics with just the name label
	customPlugin := &gm.GormMetrics{
		HistogramVec: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gorm_custom_metric",
			Help:    "Custom GORM metric with just name label",
			Buckets: prometheus.DefBuckets,
		}, []string{"name"}),
		LabelFn: func(db *gorm.DB, action gm.Action) []string {
			return []string{
				db.Statement.Context.Value(gm.GormMetricsContextKey).(*gm.MetricContextValue).Name(),
			}
		},
	}

	if err := db.Use(customPlugin); err != nil {
		t.Fatalf("failed to use custom plugin: %v", err)
	}
	if err := db.AutoMigrate(&Person{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	person := &Person{Name: "Bob", Age: 40}
	if err := db.WithContext(gm.WithName("my_create")).Create(person).Error; err != nil {
		t.Fatalf("failed to create person model: %v", err)
	}

	var value io_prometheus_client.Metric
	// verify successful create metric with custom histogram
	metric, err := customPlugin.HistogramVec.MetricVec.GetMetricWithLabelValues("my_create")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	if err := metric.Write(&value); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if value.GetHistogram() == nil {
		t.Fatalf("expected histogram metric to be recorded, but it was nil")
	}
	if value.GetHistogram().GetSampleCount() != 1 {
		t.Fatalf("expected sample count to be 1, got %d", value.GetHistogram().GetSampleCount())
	}
}
