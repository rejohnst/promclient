package rules

import (
	"context"
	"fmt"
	"os"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func Rules(ctx context.Context, api v1.API) {
	result, err := api.Rules(ctx)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of rules: %v\n", err)
		os.Exit(1)
	}

	for _, rulegrp := range result.Groups {
		fmt.Printf("Group: %s\n", rulegrp.Name)
		for _, rule := range rulegrp.Rules {
			switch v := rule.(type) {
			case v1.RecordingRule:
				recRule := rule.(v1.RecordingRule)
				fmt.Printf("\tRecording Rule: %s\n", recRule.Name)
			case v1.AlertingRule:
				alertRule := rule.(v1.AlertingRule)
				fmt.Printf("\tAlert Rule: %s\n", alertRule.Name)
			default:
				fmt.Printf("unknown rule type %s\n", v)
			}
		}
	}
}
