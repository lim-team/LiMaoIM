package lim

import "github.com/prometheus/client_golang/prometheus"

var (
	proposalsApplied = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "limao",
		Subsystem: "server",
		Name:      "proposals_applied_total",
		Help:      "The total number of consensus proposals applied.",
	})
	applySnapshotInProgress = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "limao",
		Subsystem: "server",
		Name:      "snapshot_apply_in_progress_total",
		Help:      "1 if the server is applying the incoming snapshot. 0 if none.",
	})
)
