package metrics

import (
	"context"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/logging"
)

const logMetrics = "metrics"

var log = logging.New(logMetrics)

func Exporter() http.Handler {
	exporter, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "venus_sector_manager",
	})
	if err != nil {
		log.Errorf("could not create the prometheus stats exporter: %v", err)
	}

	if err := view.Register(VenusClusterViews...); err != nil {
		panic(err)
	}

	stats.Record(context.Background(), VenusClusterInfo.M(int64(1)))
	return exporter
}
