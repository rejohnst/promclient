package query

import (
	"context"
	"fmt"
	"io"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type QueryArgs struct {
	// Timed prints query elapsed time.
	Timed bool
	// SkipTimestamp omits timestamps from query output.
	SkipTimestamp bool
}

type QueryParams struct {
	Query string
	Range v1.Range
}

// Query performs a PromQL query and prints the results.
func Query(ctx context.Context, api v1.API, out, errOut io.Writer, args *QueryArgs, params *QueryParams) error {
	start := time.Now()
	result, warnings, err := execute(ctx, api, params)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	if len(warnings) > 0 {
		fmt.Fprintf(errOut, "warnings: %v\n", warnings)
	}

	if err := printValue(out, result, args.SkipTimestamp); err != nil {
		return err
	}

	if args.Timed {
		_, err := fmt.Fprintf(out, "query time: %s\n", elapsed.Round(time.Millisecond))
		return err
	}

	return nil
}

func execute(ctx context.Context, api v1.API, params *QueryParams) (model.Value, v1.Warnings, error) {
	if (v1.Range{}) == params.Range {
		return api.Query(ctx, params.Query, time.Now())
	}
	return api.QueryRange(ctx, params.Query, params.Range)
}

func printValue(w io.Writer, value model.Value, skipTimestamp bool) error {
	if value == nil {
		return fmt.Errorf("query returned no result")
	}

	switch value := value.(type) {
	case model.Vector:
		for _, sample := range value {
			if !skipTimestamp {
				fmt.Fprintf(w, "[%s] %s = %v\n",
					sample.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
					sample.Metric.String(), sample.Value)
				continue
			}
			fmt.Fprintf(w, "%s = %v\n", sample.Metric.String(), sample.Value)
		}
	case model.Matrix:
		for _, stream := range value {
			fmt.Fprintf(w, "%s = \n", stream.Metric.String())
			for _, sample := range stream.Values {
				if !skipTimestamp {
					fmt.Fprintf(w, "\t[%s] %v\n",
						sample.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
						sample.Value)
					continue
				}
				fmt.Fprintf(w, "\t%v\n", sample.Value)
			}
		}
	case *model.Scalar:
		fmt.Fprintf(w, "%v\n", value)
	case *model.String:
		fmt.Fprintf(w, "%v\n", value)
	default:
		return fmt.Errorf("unsupported query result type %s", value.Type())
	}

	return nil
}
