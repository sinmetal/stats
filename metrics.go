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

func InitExporter(project string, location string, namespace string, node string, labels *stackdriver.Labels) *stackdriver.Exporter {
	mr := NewGenericNodeMonitoredResource(location, namespace, node)
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
	LogSize       = "logsize"
	RedisStatus   = "redis-status"
	SpannerStatus = "spanner-status"

	// Units are used to define Measures of OpenCensus.
	ByteSizeUnit = "byte"
	CountUnit    = "count"

	// ResouceNamespace is used for the exporter to have resource labels.
	ResourceNamespace = "sinmetal"
)

var (
	// Measure variables
	MLogSize            = stats.Int64(LogSize, "logSize", ByteSizeUnit)
	MRedisStatusCount   = stats.Int64(RedisStatus, "redis status", CountUnit)
	MSpannerStatusCount = stats.Int64(SpannerStatus, "spanner status", CountUnit)

	RedisStatusCountView = &view.View{
		Name:        RedisStatus,
		Description: "redis status count",
		TagKeys:     []tag.Key{KeySource},
		Measure:     MRedisStatusCount,
		Aggregation: view.Count(),
	}

	SpannerStatusCountView = &view.View{
		Name:        SpannerStatus,
		Description: "spanner status count",
		TagKeys:     []tag.Key{KeySource},
		Measure:     MSpannerStatusCount,
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
		SpannerStatusCountView,
	}

	// KeySource is the key for label in "generic_node",
	KeySource, _ = tag.NewKey("source")
)

func InitOpenCensusStats(exporter *stackdriver.Exporter) error {
	view.SetReportingPeriod(5 * time.Minute)
	view.RegisterExporter(exporter)
	if err := view.Register(LogSizeViews...); err != nil {
		return err
	}
	if err := view.Register(StatusViews...); err != nil {
		return err
	}

	return nil
}

func RecordMeasurement(id string, logSize int64) error {
	ctx, err := tag.New(context.Background(), tag.Upsert(KeySource, id))
	if err != nil {
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
		return err
	}

	stats.Record(ctx,
		MRedisStatusCount.M(1))
	return nil
}

func CountSpannerStatus(ctx context.Context, id string) error {
	ctx, err := tag.New(ctx, tag.Upsert(KeySource, id))
	if err != nil {
		return err
	}

	stats.Record(ctx,
		MSpannerStatusCount.M(1))
	return nil
}
