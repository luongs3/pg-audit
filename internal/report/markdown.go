package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/luongs3/pg-audit/internal/collector"
)

func Markdown(f *collector.Findings) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Postgres audit: `%s`\n\n", f.DatabaseName)
	fmt.Fprintf(&b, "_Generated %s by pg-audit. Postgres version: %s._\n\n",
		time.Now().UTC().Format(time.RFC3339), f.PgVersion)

	fmt.Fprintln(&b, "## Summary")
	crit, warn := count(f, collector.Critical), count(f, collector.Warning)
	fmt.Fprintf(&b, "- %d critical finding(s)\n- %d warning(s)\n- %d section(s) scanned\n\n",
		crit, warn, len(f.Sections))

	for _, s := range f.Sections {
		fmt.Fprintf(&b, "## %s\n\n", s.Name)
		if s.Skipped != "" {
			fmt.Fprintf(&b, "_Skipped: %s_\n\n", s.Skipped)
			continue
		}
		if len(s.Findings) == 0 {
			fmt.Fprintln(&b, "_No findings._")
			fmt.Fprintln(&b)
			continue
		}
		for _, x := range s.Findings {
			fmt.Fprintf(&b, "### [%s] %s\n\n%s\n\n", strings.ToUpper(string(x.Severity)), x.Title, x.Detail)
		}
	}

	fmt.Fprintln(&b, "---")
	fmt.Fprintln(&b, "Want a senior backend engineer to read this report and send back a")
	fmt.Fprintln(&b, "prioritized fix plan with rollout steps? $800 flat, 48-hour turnaround.")
	fmt.Fprintln(&b, "See README for details.")
	return b.String()
}

func count(f *collector.Findings, sev collector.Severity) int {
	return f.Count(sev)
}
