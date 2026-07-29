package main

import (
	"fmt"
	"os"

	"github.com/tonyredondo/buildopt/internal/metricscatalog"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(
			os.Stderr,
			"usage: metrics-catalog-validator <catalog.json>",
		)
		os.Exit(64)
	}
	catalog, err := metricscatalog.Load(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "metrics catalog invalid: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(
		os.Stdout,
		"METRICS-001 catalog valid: version=%s metrics=%d policy=%s\n",
		catalog.MetricDefinitionVersion,
		len(catalog.Metrics),
		catalog.PromotionPolicy.Version,
	)
}
