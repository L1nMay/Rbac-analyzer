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
	inputDir := flag.String("input-dir", "./manifests", "Directory containing RBAC YAML manifests")
	outputFormat := flag.String("output", "table", "Output format: table or json")
	dangerOnly := flag.Bool("danger-only", false, "Show only dangerous roles")
	namespaceFilter := flag.String("namespace", "", "Filter by namespace (empty for all)")

	flag.Parse()

	data, err := loader.LoadFromDir(*inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to load RBAC data: %v\n", err)
		os.Exit(1)
	}

	subjectPerms := rbac.BuildSubjectPermissions(
		data.Roles,
		data.ClusterRoles,
		data.RoleBindings,
		data.ClusterRoleBindings,
	)

	if len(subjectPerms) == 0 {
		fmt.Println("No RBAC data found (no Roles/Bindings detected).")
		return
	}

	switch *outputFormat {
	case "table":
		if err := output.PrintTable(os.Stdout, subjectPerms, *dangerOnly, *namespaceFilter); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to print table: %v\n", err)
			os.Exit(1)
		}
	case "json":
		if err := output.PrintJSON(os.Stdout, subjectPerms, *dangerOnly, *namespaceFilter); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to print json: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown output format %q (use table or json)\n", *outputFormat)
		os.Exit(1)
	}
}
