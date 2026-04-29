package rules

import (
	"context"
	"fmt"
	"io"
	"sort"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type RuleArgs struct {
	RuleName  string
	RuleGroup string
}

func printLabelSet(w io.Writer, labels model.LabelSet, indent string) {
	sortedLabels := make([]string, 0, len(labels))
	for labelName := range labels {
		sortedLabels = append(sortedLabels, string(labelName))
	}
	sort.Strings(sortedLabels)

	for _, labelName := range sortedLabels {
		fmt.Fprintf(w, "%s%s: %s\n", indent, labelName, labels[model.LabelName(labelName)])
	}
}

func printAlertRule(w io.Writer, rule v1.AlertingRule) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-20s %s\n", "Type:", "alerting rule")
	fmt.Fprintf(w, "%-20s %s\n", "Name:", rule.Name)
	fmt.Fprintf(w, "%-20s %s\n", "Expression:", rule.Query)
	fmt.Fprintf(w, "%-20s %v secs\n", "For:", rule.Duration)
	fmt.Fprintln(w, "Annotations:")
	printLabelSet(w, rule.Annotations, "  ")
	fmt.Fprintln(w, "Labels:")
	printLabelSet(w, rule.Labels, "  ")
	fmt.Fprintf(w, "%-20s %s\n", "Rule Health:", rule.Health)
	if rule.Health != "ok" {
		fmt.Fprintf(w, "    %-20s %s\n", "last error:", rule.LastError)
	}
	fmt.Fprintf(w, "%-20s %s\n", "State:", rule.State)
	if len(rule.Alerts) > 0 {
		fmt.Fprintln(w, "Active Alerts:")
		for _, alert := range rule.Alerts {
			fmt.Fprintf(w, "  %-20s %s\n", "Message:", alert.Annotations["message"])
			fmt.Fprintf(w, "  %-20s %s\n", "Last Fired:", alert.ActiveAt.Local().String())
		}
	}
	fmt.Fprintf(w, "%-20s %v secs\n", "Evaluation Time:", rule.EvaluationTime)
	fmt.Fprintf(w, "%-20s %s\n", "Last Evaluation:", rule.LastEvaluation.Local().String())
}

func printRecordRule(w io.Writer, rule v1.RecordingRule) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%-20s %s\n", "Type:", "recording rule")
	fmt.Fprintf(w, "%-20s %s\n", "Name:", rule.Name)
	fmt.Fprintf(w, "%-20s %s\n", "Expression:", rule.Query)
	fmt.Fprintln(w, "Labels:")
	printLabelSet(w, rule.Labels, "  ")
	fmt.Fprintf(w, "%-20s %s\n", "Rule Health:", rule.Health)
	if rule.Health != "ok" {
		fmt.Fprintf(w, "    %-20s %s\n", "last error:", rule.LastError)
	}
	fmt.Fprintf(w, "%-20s %v secs\n", "Evaluation Time:", rule.EvaluationTime)
	fmt.Fprintf(w, "%-20s %s\n", "Last Evaluation:", rule.LastEvaluation.Local().String())
}

func Rules(ctx context.Context, api v1.API, w io.Writer, args RuleArgs) error {
	result, err := api.Rules(ctx)
	if err != nil {
		return fmt.Errorf("retrieve rules: %w", err)
	}

	for _, ruleGroup := range result.Groups {
		if args.RuleGroup != "" && args.RuleGroup != ruleGroup.Name {
			continue
		}
		if args.RuleName == "" && (args.RuleGroup == "" || args.RuleGroup == ruleGroup.Name) {
			fmt.Fprintf(w, "Group: %s\n", ruleGroup.Name)
		}

		for _, rule := range ruleGroup.Rules {
			switch rule := rule.(type) {
			case v1.AlertingRule:
				if args.RuleName != "" {
					if args.RuleName == rule.Name {
						printAlertRule(w, rule)
					}
					continue
				}
				fmt.Fprintf(w, "\tAlert Rule: %s\n", rule.Name)
			case v1.RecordingRule:
				if args.RuleName != "" {
					if args.RuleName == rule.Name {
						printRecordRule(w, rule)
					}
					continue
				}
				fmt.Fprintf(w, "\tRecording Rule: %s\n", rule.Name)
			default:
				fmt.Fprintf(w, "unknown rule type %T\n", rule)
			}
		}
	}
	fmt.Fprintln(w)

	return nil
}
