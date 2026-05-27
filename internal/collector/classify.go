package collector

// This file isolates every severity decision the collectors make.
//
// The detectors elsewhere in this package are mostly SQL + row scanning;
// the part that actually encodes judgement — "at what threshold does a
// measurement become a warning, and when is it critical?" — lives here as
// pure functions. Keeping it separate buys two things:
//
//   1. The thresholds are testable without a live database. The decision
//      logic is the part most likely to be wrong (an off-by-one on a
//      boundary, a unit mismatch), and it's the part a reader most wants
//      to audit. A pure function with a table-driven test is the cheapest
//      way to pin it.
//   2. The thresholds are reviewable in one place instead of scattered
//      across eight query functions. Tuning "what counts as bloat" is a
//      one-line edit here, not an archaeology dig.
//
// Every threshold below is a judgement call, not a law of nature. They are
// documented inline so the next reader can disagree with a specific number
// rather than the whole tool.

const (
	mib = 1024 * 1024
	gib = 1024 * 1024 * 1024
)

// classifySlowQuery grades a statement by its mean execution time.
// Mean (not total) is used deliberately: a cheap query called millions of
// times shows up under "calls", but a query that's individually slow is the
// one a human needs to look at first.
func classifySlowQuery(meanMs float64) Severity {
	switch {
	case meanMs > 500:
		return Critical
	case meanMs > 100:
		return Warning
	default:
		return Info
	}
}

// classifyBloat grades estimated table bloat. The bool reports whether the
// finding is worth surfacing at all — anything under 25% is noise on a
// healthy table and is dropped rather than rendered as Info, to keep the
// report's signal high. Critical requires BOTH a high ratio and real
// absolute waste, so a 60%-bloated 2 MB scratch table doesn't cry wolf.
func classifyBloat(bloatPct float64, bloatBytes int64) (sev Severity, report bool) {
	switch {
	case bloatPct >= 50 && bloatBytes > 100*mib:
		return Critical, true
	case bloatPct >= 25:
		return Warning, true
	default:
		return Info, false
	}
}

// classifyMissingIndex grades a table that the planner keeps sequentially
// scanning. Critical is reserved for tables that are both heavily scanned
// and large enough that the scans actually hurt — a small table getting
// seq-scanned is usually fine (the planner often prefers it).
func classifyMissingIndex(seqScan, sizeBytes int64) Severity {
	if seqScan > 100000 && sizeBytes > 500*mib {
		return Critical
	}
	return Warning
}

// classifyDBCacheHit grades the database-wide buffer cache hit ratio (0-100).
// OLTP workloads want >99%; below 90% is almost always undersized
// shared_buffers or a cold cache.
func classifyDBCacheHit(ratioPct float64) Severity {
	switch {
	case ratioPct < 90:
		return Critical
	case ratioPct < 99:
		return Warning
	default:
		return Info
	}
}

// classifyTableCacheHit grades a single hot table's cache hit ratio (0-100).
// The bar is lower than the DB-wide check: one table reaching for disk is
// often the largest table in the system and expected, so only <95% is
// flagged, and only as a Warning.
func classifyTableCacheHit(hitPct float64) Severity {
	if hitPct < 95 {
		return Warning
	}
	return Info
}

// classifyUnusedIndex grades a single never-scanned index by its size.
// Small unused indexes are cheap to keep; large ones waste disk and slow
// every write to the table.
func classifyUnusedIndex(sizeBytes int64) Severity {
	if sizeBytes > 100*mib {
		return Warning
	}
	return Info
}

// unusedIndexHeadline decides whether the unused-index section earns a
// Critical headline summarising total reclaimable space. The bool reports
// whether the headline should be prepended at all.
func unusedIndexHeadline(totalBytes int64) (sev Severity, show bool) {
	if totalBytes > gib {
		return Critical, true
	}
	return Info, false
}

// classifyReplicationLag grades a replica by replay lag (seconds) and bytes
// behind the primary. Either dimension can independently push it Critical:
// a replica that's only 5s behind but 200 MB behind in WAL is as much a
// problem as one that's 90s behind.
func classifyReplicationLag(replayLagSec float64, lagBytes int64) Severity {
	switch {
	case replayLagSec > 60 || lagBytes > 100*mib:
		return Critical
	case replayLagSec > 10:
		return Warning
	default:
		return Info
	}
}

// classifyIdleInTxn grades an idle-in-transaction session by its age in
// seconds. These hold locks and block VACUUM, so a long one is Critical.
func classifyIdleInTxn(ageSec int) Severity {
	if ageSec > 600 {
		return Critical
	}
	return Warning
}

// classifyConnectionUsage grades how close the server is to max_connections.
// Running out of connection slots causes hard "too many clients" failures, so
// the thresholds are deliberately conservative: most setups should sit well
// below 80% steady-state, and crossing 90% means a saturation incident is one
// traffic spike away.
func classifyConnectionUsage(usedPct float64) Severity {
	switch {
	case usedPct > 90:
		return Critical
	case usedPct > 80:
		return Warning
	default:
		return Info
	}
}

// classifyMissingPrimaryKey grades a table that lacks a primary key. It's a
// real schema-health issue (logical replication, row addressing, dedupe) but
// not an availability risk, so it's a Warning rather than Critical.
func classifyMissingPrimaryKey() Severity {
	return Warning
}
