package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONIsValidAndCountsSeverities(t *testing.T) {
	out, err := JSON(sampleFindings())
	if err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}

	summary, ok := got["summary"].(map[string]any)
	if !ok {
		t.Fatalf("missing summary object in %v", got)
	}
	if summary["critical"].(float64) != 1 {
		t.Errorf("expected 1 critical, got %v", summary["critical"])
	}
	if summary["warning"].(float64) != 1 {
		t.Errorf("expected 1 warning, got %v", summary["warning"])
	}
	if summary["sections"].(float64) != 3 {
		t.Errorf("expected 3 sections, got %v", summary["sections"])
	}
	if got["database"] != "shop" {
		t.Errorf("expected database=shop, got %v", got["database"])
	}
}

func TestJSONOmitsEmptyEvidenceAndRendersSkipped(t *testing.T) {
	out, err := JSON(sampleFindings())
	if err != nil {
		t.Fatal(err)
	}
	// The skipped section should carry its skipped reason.
	if !strings.Contains(out, `"skipped": "extension not installed"`) {
		t.Errorf("expected skipped reason in JSON output:\n%s", out)
	}
	// Findings without evidence should omit the key, not render "evidence": null.
	if strings.Contains(out, `"evidence": null`) {
		t.Errorf("nil evidence should be omitted, not rendered as null:\n%s", out)
	}
}
