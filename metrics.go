package stats

import (
	"context"
	"fmt"
	"log"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type GenericNodeMonitoredResource struct {
	Location    string
	NamespaceId string
	NodeId      string
}

func NewGenericNodeMonitoredResource(location, namespace, node string) *GenericNodeMonitoredResource {
	return &GenericNodeMonitoredResource{
		Location:    location,
		NamespaceId: namespace,
		NodeId:      node,
	}
}

func (mr *GenericNodeMonitoredResource) MonitoredResource() (string, map[string]string) {
	labels := map[string]string{
		"location":  mr.Location,
		"namespace": mr.NamespaceId,
		"node_id":   mr.NodeId,
	}
	return "generic_node", labels
}

func GetMetricType(v *view.View) string {
	return fmt.Sprintf("custom.googleapis.com/%s", v.Name)
}

func InitExporter(project string) *stackdriver.Exporter {
	location := "asia-northeast1-b" // TODO 適当に入れてる

	mr := NewGenericNodeMonitoredResource(location, "default", "public-data")
	labels := &stackdriver.Labels{}
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               project,
		Location:                location,
		MonitoredResource:       mr,
		DefaultMonitoringLabels: labels,
		GetMetricType:           GetMetricType,
	})
	if err != nil {
		log.Fatal("failed to initialize ")
	}
	return exporter
}

const (
	// OCReportInterval is the interval for OpenCensus to send stats data to
	// Stackdriver Monitoring via its exporter.
	// NOTE: this value should not be no less than 1 minute. Detailes are in the doc.
	// https://cloud.google.com/monitoring/custom-metrics/creating-metrics#writing-ts
	OCReportInterval = 60 * time.Second

	// Measure namess for respecitive OpenCensus Measure
	LogSize     = "logsize"
	RedisStatus = "redis-status"

	// Units are used to define Measures of OpenCensus.
	ByteSizeUnit = "byte"
	StatusUnit   = "count"

	// ResouceNamespace is used for the exporter to have resource labels.
	ResourceNamespace = "sinmetal"
)

var (
	// Measure variables
	MLogSize     = stats.Int64(LogSize, "logSize", ByteSizeUnit)
	MStatusCount = stats.Int64(RedisStatus, "redis status", StatusUnit)

	RedisStatusCountView = &view.View{
		Name:        RedisStatus,
		Description: "status count",
		TagKeys:     []tag.Key{KeySource},
		Measure:     MStatusCount,
		Aggregation: view.Count(),
	}

	LogSizeView = &view.View{
		Name:        LogSize,
		Measure:     MLogSize,
		TagKeys:     []tag.Key{KeySource},
		Description: "log size",
		Aggregation: view.Sum(),
	}

	LogSizeViews = []*view.View{
		LogSizeView,
	}

	StatusViews = []*view.View{
		RedisStatusCountView,
	}

	// KeySource is the key for label in "generic_node",
	KeySource, _ = tag.NewKey("source")
)

func InitOpenCensusStats(exporter *stackdriver.Exporter) {
	view.SetReportingPeriod(5 * time.Minute)
	view.RegisterExporter(exporter)
	view.Register(LogSizeViews...)
	if err := view.Register(StatusViews...); err != nil {
		log.Fatal(err)
	}
}

func RecordMeasurement(id string, logSize int64) error {
	ctx, err := tag.New(context.Background(), tag.Upsert(KeySource, id))
	if err != nil {
		log.Fatalf("failed to insert key: %v", err)
		return err
	}

	stats.Record(ctx,
		MLogSize.M(logSize),
	)
	return nil
}

func CountRedisStatus(ctx context.Context, id string) error {
	ctx, err := tag.New(ctx, tag.Upsert(KeySource, id))
	if err != nil {
		log.Fatalf("failed to insert key: %v", err)
		return err
	}

	stats.Record(ctx,
		MStatusCount.M(1))
	return nil
}