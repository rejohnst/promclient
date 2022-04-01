package query

import (
	"context"
	"fmt"
	"os"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type QueryArgs struct {
	// if true, print query time elapsed
	Timed bool
}

type QueryParams struct {
	Query string
	Range v1.Range
}

//
// Perform a PromQL query and print the results
//
func Query(ctx context.Context, api v1.API, args *QueryArgs, pq *QueryParams, skipTimestamp *bool) {
	var result model.Value
	var warnings v1.Warnings
	var err error
	var t1, t2 time.Time

	t1 = time.Now()
	if (v1.Range{}) == pq.Range {
		result, warnings, err = api.Query(ctx, pq.Query, time.Now())
	} else {
		result, warnings, err = api.QueryRange(ctx, pq.Query, pq.Range)
	}
	t2 = time.Now()

	if err != nil {
		fmt.Fprintf(os.Stderr, "query failed: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "warnings: %v\n", warnings)
	}

	//
	// result will contain a model.Value, which is a generic interface, so we
	// use a type switch to process it.  Instant queries should return type
	// model.Vector and range queries should return type model.Matrix.
	//
	switch {
	case result.Type() == model.ValVector:
		samples := result.(model.Vector)
		for _, sample := range samples {
			if !*skipTimestamp {
				fmt.Printf("[%s] %s = %v\n",
					sample.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
					sample.Metric.String(), sample.Value)
			} else {
				fmt.Printf("%s = %v\n",
					sample.Metric.String(), sample.Value)
			}
		}
	case result.Type() == model.ValMatrix:
		streams := result.(model.Matrix)
		for _, stream := range streams {
			fmt.Printf("%s = \n", stream.Metric.String())
			for _, val := range stream.Values {
				if !*skipTimestamp {
					fmt.Printf("\t[%s] %v\n",
						val.Timestamp.Time().Format("Jan 2 15:04:05 -0700 MST"),
						val.Value)
				} else {
					fmt.Printf("\t%v\n",
						val.Value)
				}

			}
		}
	case result.Type() == model.ValScalar:
		fmt.Printf("%v\n", result)
	case result.Type() == model.ValString:
		fmt.Printf("%v\n", result)
	}

	if args.Timed {
		elapsed := t2.Sub(t1)
		// time.Format is weird - see https://golang.org/pkg/time/#Time.Format
		fmt.Printf("query time: %s\n", time.Time{}.Add(elapsed).Format("04:05"))
	}
}
