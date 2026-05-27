package collector

import "testing"

func TestFindingsCount(t *testing.T) {
	f := &Findings{
		Sections: []Section{
			{
				Name: "Cache hit ratio",
				Findings: []Finding{
					{Severity: Critical, Title: "low cache hit"},
				},
			},
			{
				Name: "Unused indexes",
				Findings: []Finding{
					{Severity: Warning, Title: "idx_a unused"},
					{Severity: Warning, Title: "idx_b unused"},
					{Severity: Info, Title: "fyi"},
				},
			},
			{Name: "Replication lag", Skipped: "not a primary"},
		},
	}

	if got := f.Count(Critical); got != 1 {
		t.Errorf("Count(Critical) = %d, want 1", got)
	}
	if got := f.Count(Warning); got != 2 {
		t.Errorf("Count(Warning) = %d, want 2", got)
	}
	if got := f.Count(Info); got != 1 {
		t.Errorf("Count(Info) = %d, want 1", got)
	}
}

func TestFindingsCountEmpty(t *testing.T) {
	f := &Findings{}
	if got := f.Count(Critical); got != 0 {
		t.Errorf("Count on empty findings = %d, want 0", got)
	}
}
