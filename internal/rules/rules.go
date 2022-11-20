package rules

import (
	"context"
	"fmt"
	"os"
	"sort"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type RuleArgs struct {
	RuleName  string
	RuleGroup string
}

//
// Print a model.LabelsSet in sorted (by keys) orderm with fields
// optionally indentex
//
func printLabelSet(labels model.LabelSet, indent string) {
	sortedLabels := make([]string, 0, len(labels))
	for labelname := range labels {
		sortedLabels = append(sortedLabels, string(labelname))
	}

	sort.Strings(sortedLabels)

	for _, labelname := range sortedLabels {
		fmt.Printf("%s%s: %s\n", indent, labelname, labels[model.LabelName(labelname)])
	}
}

func printAlertRule(rule v1.AlertingRule) {
	fmt.Printf("\n")
	fmt.Printf("%-20s %s\n", "Type:", "alerting rule")
	fmt.Printf("%-20s %s\n", "Name:", rule.Name)
	fmt.Printf("%-20s %s\n", "Expression:", rule.Query)
	fmt.Printf("%-20s %v secs\n", "For:", rule.Duration)
	fmt.Printf("Annotations:\n")
	printLabelSet(rule.Annotations, "  ")
	fmt.Printf("Labels:\n")
	printLabelSet(rule.Labels, "  ")
	fmt.Printf("%-20s %s\n", "Rule Health:", rule.Health)
	if rule.Health != "ok" {
		fmt.Printf("    %-20s %s\n", "last error:", rule.LastError)
	}
	fmt.Printf("%-20s %s\n", "State:", rule.State)
	if len(rule.Alerts) > 0 {
		fmt.Printf("Active Alerts:\n")
		for _, alert := range rule.Alerts {
			fmt.Printf("  %-20s %s\n", "Message:", alert.Annotations["message"])
			fmt.Printf("  %-20s %s\n", "Last Fired:", alert.ActiveAt.Local().String())
		}
	}
	fmt.Printf("%-20s %v secs\n", "Evaluation Time:", rule.EvaluationTime)
	fmt.Printf("%-20s %s\n", "Last Evaluation:", rule.LastEvaluation.Local().String())
}

func printRecordRule(rule v1.RecordingRule) {
	fmt.Printf("\n")
	fmt.Printf("%-20s %s\n", "Type:", "recording rule")
	fmt.Printf("%-20s %s\n", "Name:", rule.Name)
	fmt.Printf("%-20s %s\n", "Expression:", rule.Query)
	fmt.Printf("Labels:\n")
	printLabelSet(rule.Labels, "  ")
	fmt.Printf("%-20s %s\n", "Rule Health:", rule.Health)
	if rule.Health != "ok" {
		fmt.Printf("    %-20s %s\n", "last error:", rule.LastError)
	}
	fmt.Printf("%-20s %v secs\n", "Evaluation Time:", rule.EvaluationTime)
	fmt.Printf("%-20s %s\n", "Last Evaluation:", rule.LastEvaluation.Local().String())
}

func Rules(ctx context.Context, api v1.API, args RuleArgs) {
	result, err := api.Rules(ctx)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving list of rules: %v\n", err)
		os.Exit(1)
	}

	for _, rulegrp := range result.Groups {
		if args.RuleGroup != "" && args.RuleGroup != rulegrp.Name {
			continue
		}
		if args.RuleName == "" &&
			(args.RuleGroup == "" || args.RuleGroup == rulegrp.Name) {
			fmt.Printf("Group: %s\n", rulegrp.Name)
		}
		for _, rule := range rulegrp.Rules {
			switch v := rule.(type) {
			case v1.AlertingRule:
				alertRule := rule.(v1.AlertingRule)
				if args.RuleName != "" {
					if args.RuleName == alertRule.Name {
						printAlertRule(alertRule)
					}
				} else {
					fmt.Printf("\tAlert Rule: %s\n", alertRule.Name)
				}
			case v1.RecordingRule:
				recRule := rule.(v1.RecordingRule)
				if args.RuleName != "" {
					if args.RuleName == recRule.Name {
						printRecordRule(recRule)
					}
				} else {
					fmt.Printf("\tRecording Rule: %s\n", recRule.Name)
				}
			default:
				fmt.Printf("unknown rule type %s\n", v)
			}
		}
	}
	fmt.Printf("\n")
}
