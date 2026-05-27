package collector

import "testing"

// These tests pin the severity thresholds. The interesting cases are the
// boundaries — the exact value where Info becomes Warning becomes Critical —
// because that's where an off-by-one or a >/>= slip actually changes the
// report a user sees. Each table therefore brackets every threshold with a
// just-below / exact / just-above triple.

func TestClassifySlowQuery(t *testing.T) {
	tests := []struct {
		name   string
		meanMs float64
		want   Severity
	}{
		{"zero", 0, Info},
		{"just below warning", 100, Info}, // boundary is > 100, so 100 itself is Info
		{"just above warning", 100.01, Warning},
		{"mid warning", 250, Warning},
		{"just below critical", 500, Warning}, // boundary is > 500, so 500 itself is Warning
		{"just above critical", 500.01, Critical},
		{"deep critical", 5000, Critical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifySlowQuery(tt.meanMs); got != tt.want {
				t.Errorf("classifySlowQuery(%v) = %q, want %q", tt.meanMs, got, tt.want)
			}
		})
	}
}

func TestClassifyBloat(t *testing.T) {
	tests := []struct {
		name       string
		bloatPct   float64
		bloatBytes int64
		wantSev    Severity
		wantReport bool
	}{
		{"trivial bloat suppressed", 10, 5 * mib, Info, false},
		{"just below report threshold", 24.9, 200 * mib, Info, false},
		{"exactly at warning", 25, 1 * mib, Warning, true},
		{"warning ratio but small bytes", 60, 50 * mib, Warning, true}, // high % but < 100 MB → not critical
		{"critical needs both", 50, 100 * mib, Warning, true},          // bytes must be > 100 MB, not ==
		{"critical", 50, 101 * mib, Critical, true},
		{"deep critical", 90, 5 * gib, Critical, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sev, report := classifyBloat(tt.bloatPct, tt.bloatBytes)
			if sev != tt.wantSev || report != tt.wantReport {
				t.Errorf("classifyBloat(%v, %d) = (%q, %v), want (%q, %v)",
					tt.bloatPct, tt.bloatBytes, sev, report, tt.wantSev, tt.wantReport)
			}
		})
	}
}

func TestClassifyMissingIndex(t *testing.T) {
	tests := []struct {
		name      string
		seqScan   int64
		sizeBytes int64
		want      Severity
	}{
		{"small table heavy scan", 1_000_000, 10 * mib, Warning}, // small table → never critical
		{"big table light scan", 1000, 5 * gib, Warning},         // few scans → never critical
		{"scan boundary", 100000, 600 * mib, Warning},            // scan must be > 100k, not ==
		{"size boundary", 100001, 500 * mib, Warning},            // size must be > 500 MB, not ==
		{"both exceeded", 100001, 501 * mib, Critical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyMissingIndex(tt.seqScan, tt.sizeBytes); got != tt.want {
				t.Errorf("classifyMissingIndex(%d, %d) = %q, want %q",
					tt.seqScan, tt.sizeBytes, got, tt.want)
			}
		})
	}
}

func TestClassifyDBCacheHit(t *testing.T) {
	tests := []struct {
		name  string
		ratio float64
		want  Severity
	}{
		{"terrible", 50, Critical},
		{"just below critical bound", 89.99, Critical},
		{"exactly 90 is warning", 90, Warning}, // boundary is < 90 for critical
		{"mid warning", 95, Warning},
		{"just below healthy", 98.99, Warning},
		{"exactly 99 is healthy", 99, Info}, // boundary is < 99 for warning
		{"perfect", 100, Info},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyDBCacheHit(tt.ratio); got != tt.want {
				t.Errorf("classifyDBCacheHit(%v) = %q, want %q", tt.ratio, got, tt.want)
			}
		})
	}
}

func TestClassifyTableCacheHit(t *testing.T) {
	tests := []struct {
		name   string
		hitPct float64
		want   Severity
	}{
		{"low", 80, Warning},
		{"just below bound", 94.99, Warning},
		{"exactly 95 is info", 95, Info}, // boundary is < 95 for warning
		{"healthy", 99.9, Info},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyTableCacheHit(tt.hitPct); got != tt.want {
				t.Errorf("classifyTableCacheHit(%v) = %q, want %q", tt.hitPct, got, tt.want)
			}
		})
	}
}

func TestClassifyUnusedIndex(t *testing.T) {
	tests := []struct {
		name      string
		sizeBytes int64
		want      Severity
	}{
		{"small", 2 * mib, Info},
		{"size boundary", 100 * mib, Info}, // boundary is > 100 MB, so == is Info
		{"large", 101 * mib, Warning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyUnusedIndex(tt.sizeBytes); got != tt.want {
				t.Errorf("classifyUnusedIndex(%d) = %q, want %q", tt.sizeBytes, got, tt.want)
			}
		})
	}
}

func TestUnusedIndexHeadline(t *testing.T) {
	tests := []struct {
		name       string
		totalBytes int64
		wantSev    Severity
		wantShow   bool
	}{
		{"below threshold", 500 * mib, Info, false},
		{"exactly 1 GB", gib, Info, false}, // boundary is > 1 GB, so == does not show
		{"above threshold", gib + 1, Critical, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sev, show := unusedIndexHeadline(tt.totalBytes)
			if sev != tt.wantSev || show != tt.wantShow {
				t.Errorf("unusedIndexHeadline(%d) = (%q, %v), want (%q, %v)",
					tt.totalBytes, sev, show, tt.wantSev, tt.wantShow)
			}
		})
	}
}

func TestClassifyReplicationLag(t *testing.T) {
	tests := []struct {
		name      string
		replaySec float64
		lagBytes  int64
		want      Severity
	}{
		{"healthy", 0, 0, Info},
		{"just below warning", 10, 1 * mib, Info}, // boundary is > 10s
		{"just above warning", 10.01, 1 * mib, Warning},
		{"seconds critical", 61, 0, Critical},
		{"bytes critical even if seconds low", 2, 101 * mib, Critical}, // either dimension trips it
		{"bytes boundary not yet critical", 2, 100 * mib, Info},        // must be > 100 MB
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyReplicationLag(tt.replaySec, tt.lagBytes); got != tt.want {
				t.Errorf("classifyReplicationLag(%v, %d) = %q, want %q",
					tt.replaySec, tt.lagBytes, got, tt.want)
			}
		})
	}
}

func TestClassifyIdleInTxn(t *testing.T) {
	tests := []struct {
		name string
		age  int
		want Severity
	}{
		{"short", 60, Warning},
		{"boundary", 600, Warning}, // boundary is > 600s
		{"long", 601, Critical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyIdleInTxn(tt.age); got != tt.want {
				t.Errorf("classifyIdleInTxn(%d) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestClassifyConnectionUsage(t *testing.T) {
	tests := []struct {
		name    string
		usedPct float64
		want    Severity
	}{
		{"low", 10, Info},
		{"moderate", 50, Info},
		{"at warning boundary", 80, Info}, // boundary is > 80, so 80 itself is Info
		{"just above warning", 80.01, Warning},
		{"mid warning", 85, Warning},
		{"at critical boundary", 90, Warning}, // boundary is > 90, so 90 itself is Warning
		{"just above critical", 90.01, Critical},
		{"saturated", 100, Critical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyConnectionUsage(tt.usedPct); got != tt.want {
				t.Errorf("classifyConnectionUsage(%v) = %q, want %q", tt.usedPct, got, tt.want)
			}
		})
	}
}

func TestClassifyMissingPrimaryKey(t *testing.T) {
	if got := classifyMissingPrimaryKey(); got != Warning {
		t.Errorf("classifyMissingPrimaryKey() = %q, want %q", got, Warning)
	}
}
