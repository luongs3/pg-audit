package main

import (
	"context"
	"fmt"
	"os"

	"github.com/luongs3/pg-audit/internal/collector"
	"github.com/luongs3/pg-audit/internal/report"
	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	root := &cobra.Command{
		Use:   "pg-audit",
		Short: "Read-only Postgres health check — outputs a markdown report.",
	}

	var dsn, out, format string
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run all checks against a database and write a report.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dsn == "" {
				dsn = os.Getenv("PGAUDIT_DSN")
			}
			if dsn == "" {
				return fmt.Errorf("--dsn or $PGAUDIT_DSN required")
			}

			var render func(*collector.Findings) (string, error)
			switch format {
			case "markdown", "md", "":
				render = func(f *collector.Findings) (string, error) { return report.Markdown(f), nil }
			case "json":
				render = report.JSON
			default:
				return fmt.Errorf("unknown --format %q (want: markdown, json)", format)
			}

			ctx := context.Background()
			findings, err := collector.RunAll(ctx, dsn)
			if err != nil {
				return err
			}
			doc, err := render(findings)
			if err != nil {
				return err
			}
			if out == "" {
				fmt.Print(doc)
				return nil
			}
			return os.WriteFile(out, []byte(doc), 0644)
		},
	}
	runCmd.Flags().StringVar(&dsn, "dsn", "", "Postgres connection string (or set $PGAUDIT_DSN)")
	runCmd.Flags().StringVarP(&out, "out", "o", "", "output file (default: stdout)")
	runCmd.Flags().StringVarP(&format, "format", "f", "markdown", "output format: markdown or json")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}

	root.AddCommand(runCmd, versionCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
