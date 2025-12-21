package main

import (
	"flag"
	"fmt"
	"os"

	"rbac-analyzer/internal/loader"
	"rbac-analyzer/internal/output"
	"rbac-analyzer/internal/rbac"
)

func main() {
	// === FLAGS ===
	inputDir := flag.String("input-dir", "", "Directory with RBAC YAML manifests")
	outputFmt := flag.String("output", "table", "Output format: table|json")
	dangerOnly := flag.Bool("danger-only", false, "Show only dangerous permissions")
	title := flag.String("title", "RBAC Analysis Report", "Report title")

	flag.Parse()

	// === VALIDATION ===
	if *inputDir == "" {
		fmt.Fprintln(os.Stderr, "error: -input-dir is required")
		os.Exit(1)
	}

	// === LOAD RBAC ===
	data, err := loader.LoadFromDir(*inputDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load error:", err)
		os.Exit(1)
	}

	// === ANALYZE ===
	subjectPerms := rbac.BuildSubjectPermissions(
		data.Roles,
		data.ClusterRoles,
		data.RoleBindings,
		data.ClusterRoleBindings,
	)

	// === OUTPUT ===
	switch *outputFmt {
	case "table":
		output.PrintTable(
			os.Stdout,
			subjectPerms,
			*dangerOnly,
			*title,
		)
	case "json":
		output.PrintJSON(
			os.Stdout,
			subjectPerms,
			*dangerOnly,
			*title,
		)
	default:
		fmt.Fprintln(os.Stderr, "unknown output format:", *outputFmt)
		os.Exit(1)
	}
}
