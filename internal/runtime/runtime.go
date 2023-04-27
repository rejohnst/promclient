package runtime

import (
	"context"
	"fmt"
	"os"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func Runtime(ctx context.Context, api v1.API) {
	result, err := api.Runtimeinfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving Prometheus runtime info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%-30s %s\n", "CWD:", result.CWD)
	fmt.Printf("%-30s %v\n", "Last Cfg Reload Successful?:", result.ReloadConfigSuccess)
	fmt.Printf("%-30s %v\n", "Last Cfg Time:", result.LastConfigTime)
	fmt.Printf("%-30s %d\n", "# of Chunks", result.ChunkCount)
	fmt.Printf("%-30s %d\n", "# of Time Series", result.TimeSeriesCount)
	fmt.Printf("%-30s %d\n", "# of Corruptions", result.CorruptionCount)
	fmt.Printf("%-30s %d\n", "# of Go Routines", result.GoroutineCount)
	fmt.Printf("%-30s %d\n", "GOMAXPROCS", result.GOMAXPROCS)
	fmt.Printf("%-30s %s\n", "GOGC", result.GOGC)
	fmt.Printf("%-30s %s\n", "GODEBUG", result.GODEBUG)
	fmt.Printf("%-30s %s\n", "Data Retention", result.StorageRetention)
}
