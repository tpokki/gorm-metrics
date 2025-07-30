# GORM Metrics

GORM Metrics is a plugin for [GORM](https://gorm.io/) that automatically collects and exposes database operation duration metrics for monitoring and observability. It integrates with Prometheus and is designed for easy use in Go applications using GORM.


## Features

- Tracks duration of GORM operations (query, create, update, delete, raw, row)
- Prometheus histogram metrics with rich labels: action, model, joins, outcome
- Identifies number of joins in queries
- Simple integration with GORM


## Installation

Install the package using `go get`:

```sh
go get github.com/tpokki/gorm-metrics
```

## Usage

Import and register the plugin with your GORM DB instance:

```go
import (
    "github.com/tpokki/gorm-metrics"
    "gorm.io/gorm"
)

db, err := gorm.Open(...)
if err != nil {
    // handle error
}

// Register the metrics plugin
if err := db.Use(gm.Default()); err != nil {
    panic(err)
}
```


## Metrics Exposed

The plugin exposes a Prometheus histogram metric:

- `gorm_metrics_duration_seconds`: Duration of GORM operations in seconds

Labels:
- `action`: The type of GORM operation (`query`, `create`, `update`, `delete`, `raw`, `row`)
- `model`: The table/model name (e.g. `people`, `favorite_colors`)
- `joins`: Number of joins in the query (as a string, e.g. `0`, `1`)
- `outcome`: The result of the operation (`success`, `error`)

gorm_metrics_duration_seconds{action="query",model="people",joins="1",outcome="success"} 0.001

Example:

```
gorm_metrics_duration_seconds{name="default",action="query",model="people",joins="1",outcome="success"} 0.001
gorm_metrics_duration_seconds{name="my_update",action="update",model="things",joins="0",outcome="success"} 0.002
```


## Prometheus Integration

To expose metrics for Prometheus, use the [prometheus/client_golang](https://github.com/prometheus/client_golang) package and register the metrics endpoint in your HTTP server:

```go
import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":8080", nil)
```

## Testing

See `plugin_test.go` for example usage and metric assertions. The test covers plugin registration, GORM operations, and metric validation.


## Configuration

The default plugin uses Prometheus default buckets and automatically registers the histogram with the following labels:

- `action`, `model`, `joins`, `outcome`


You can customize label extraction by providing your own `LabelFn` when creating a `GormMetrics` instance:

```go
plugin := &gm.GormMetrics{
    HistogramVec: prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "gorm_custom_metric",
            Help:    "Custom GORM metric with just name label",
            Buckets: prometheus.DefBuckets,
        },
        []string{"name"}, // Only the name label
    ),
    LabelFn: func(db *gorm.DB, action gm.Action) []string {
        ctxVal, ok := db.Statement.Context.Value(gm.GormMetricsContextKey).(*gm.MetricContextValue)
        if ok {
            return []string{ctxVal.Name()}
        }
        return []string{"default"}
    },
}

// Usage example:
db.WithContext(gm.WithName("my_create")).Create(&Person{Name: "Bob", Age: 40})
```

See the code for details on advanced configuration and more examples.

## License

MIT