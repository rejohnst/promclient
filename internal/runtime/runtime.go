package runtime

import (
	"context"
	"fmt"
	"io"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func Runtime(ctx context.Context, api v1.API, w io.Writer) error {
	result, err := api.Runtimeinfo(ctx)
	if err != nil {
		return fmt.Errorf("retrieve runtime info: %w", err)
	}

	fmt.Fprintf(w, "%-30s %s\n", "CWD:", result.CWD)
	fmt.Fprintf(w, "%-30s %v\n", "Start Time:", result.StartTime)
	fmt.Fprintf(w, "%-30s %v\n", "Last Cfg Reload Successful?:", result.ReloadConfigSuccess)
	fmt.Fprintf(w, "%-30s %v\n", "Last Cfg Time:", result.LastConfigTime)
	fmt.Fprintf(w, "%-30s %d\n", "# of Corruptions", result.CorruptionCount)
	fmt.Fprintf(w, "%-30s %d\n", "# of Go Routines", result.GoroutineCount)
	fmt.Fprintf(w, "%-30s %d\n", "GOMAXPROCS", result.GOMAXPROCS)
	fmt.Fprintf(w, "%-30s %s\n", "GOGC", result.GOGC)
	fmt.Fprintf(w, "%-30s %s\n", "GODEBUG", result.GODEBUG)
	fmt.Fprintf(w, "%-30s %s\n", "Data Retention", result.StorageRetention)

	return nil
}
