// Package promapi reaches parts of the Prometheus HTTP API that the typed
// client_golang API does not expose.
package promapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

const targetsEndpoint = "/api/v1/targets"

// maxErrorDetail bounds how much of an unexpected response body is echoed back
// in an error message.
const maxErrorDetail = 512

// apiEnvelope is the wrapper Prometheus puts around every API response. The
// typed v1 client unwraps this internally, but that logic is unexported.
type apiEnvelope struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings"`
}

// FetchActiveTargets retrieves active targets. It sends state=active, which
// v1.API.Targets cannot do because it accepts no parameters and so always asks
// for every state at once. Dropped targets are far bulkier than active ones,
// and collecting them can exceed the 2 GiB gRPC message limit behind Thanos
// Query and fail the whole request.
//
// Warnings are returned rather than discarded: Thanos Query reports a failed
// store fan-out as an empty result plus warnings, which is otherwise
// indistinguishable from a Prometheus that has no targets configured.
func FetchActiveTargets(ctx context.Context, client api.Client) ([]v1.ActiveTarget, []string, error) {
	u := client.URL(targetsEndpoint, nil)
	u.RawQuery = url.Values{"state": []string{"active"}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build targets request: %w", err)
	}

	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieve targets: %w", err)
	}

	data, warnings, err := unwrap(resp, body)
	if err != nil {
		return nil, warnings, err
	}

	var result v1.TargetsResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, warnings, fmt.Errorf("decode targets: %w", err)
	}

	return result.Active, warnings, nil
}

// PrintWarnings reports API warnings so that a partial response is not
// mistaken for an empty one.
func PrintWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "warning: %s\n", warning)
	}
}

// unwrap validates the HTTP status and returns the "data" member of a
// successful API response along with any warnings.
func unwrap(resp *http.Response, body []byte) (json.RawMessage, []string, error) {
	code := resp.StatusCode
	// Prometheus reports API-level errors as a JSON body under these two
	// codes; anything else non-2xx is not worth parsing.
	apiError := code == http.StatusBadRequest || code == http.StatusUnprocessableEntity
	if code/100 != 2 && !apiError {
		return nil, nil, fmt.Errorf("retrieve targets: unexpected response %d: %s", code, detail(body))
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, fmt.Errorf("decode targets response: %w", err)
	}
	if envelope.Status == "error" {
		return nil, envelope.Warnings, fmt.Errorf("retrieve targets: %s: %s", envelope.ErrorType, envelope.Error)
	}
	if apiError {
		return nil, envelope.Warnings, fmt.Errorf("retrieve targets: inconsistent body for response code %d", code)
	}

	return envelope.Data, envelope.Warnings, nil
}

func detail(body []byte) string {
	if len(body) > maxErrorDetail {
		return string(body[:maxErrorDetail]) + "..."
	}
	return string(body)
}
