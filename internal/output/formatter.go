package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"rbac-analyzer/internal/rbac"
)

// PrintTable — человекочитаемый табличный вывод.
func PrintTable(
	w io.Writer,
	subjectPerms rbac.SubjectPermissions,
	dangerOnly bool,
	namespaceFilter string,
) error {
	nsFilter := strings.TrimSpace(namespaceFilter)

	// Отсортируем субъектов для стабильного вывода
	subjects := make([]rbac.SubjectRef, 0, len(subjectPerms))
	for s := range subjectPerms {
		subjects = append(subjects, s)
	}
	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].String() < subjects[j].String()
	})

	for _, s := range subjects {
		roles := filterRoles(subjectPerms[s], dangerOnly, nsFilter)
		if len(roles) == 0 {
			continue
		}

		fmt.Fprintf(w, "=== %s ===\n", s.String())
		for _, r := range roles {
			scope := "namespace"
			if r.ClusterScope {
				scope = "cluster"
			}
			ns := r.SourceNamespace
			if ns == "" {
				ns = "-"
			}

			dangerMark := ""
			if r.Dangerous {
				dangerMark = " [DANGEROUS]"
			}

			fmt.Fprintf(
				w,
				"  Role: %s/%s (source=%s, via=%s/%s)%s\n",
				ns,
				r.SourceName,
				r.SourceKind,
				r.BoundVia,
				r.BindingName,
				dangerMark,
			)
			fmt.Fprintf(w, "    Scope: %s\n", scope)
			if len(r.DangerReasons) > 0 {
				fmt.Fprintf(w, "    Danger reasons:\n")
				for _, reason := range r.DangerReasons {
					fmt.Fprintf(w, "      - %s\n", reason)
				}
			}
			fmt.Fprintf(w, "    Permissions:\n")
			for _, p := range r.Permissions {
				nsInfo := p.Namespace
				if p.ClusterScope {
					nsInfo = "*"
				} else if nsInfo == "" {
					nsInfo = "-"
				}
				names := ""
				if len(p.ResourceNames) > 0 {
					names = fmt.Sprintf(" names=%v", p.ResourceNames)
				}
				fmt.Fprintf(
					w,
					"      - ns=%s verb=%s resource=%s apiGroup=%s%s\n",
					nsInfo,
					p.Verb,
					p.Resource,
					p.APIGroup,
					names,
				)
			}
			fmt.Fprintln(w)
		}

		fmt.Fprintln(w)
	}

	return nil
}

// PrintJSON — JSON вывод для дальнейшей обработки.
func PrintJSON(
	w io.Writer,
	subjectPerms rbac.SubjectPermissions,
	dangerOnly bool,
	namespaceFilter string,
) error {
	nsFilter := strings.TrimSpace(namespaceFilter)

	// Строим структуру для сериализации
	type outputStruct struct {
		Subject string               `json:"subject"`
		Roles   []rbac.EffectiveRole `json:"roles"`
	}

	var out []outputStruct

	for s, roles := range subjectPerms {
		fl := filterRoles(roles, dangerOnly, nsFilter)
		if len(fl) == 0 {
			continue
		}
		out = append(out, outputStruct{
			Subject: s.String(),
			Roles:   fl,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func filterRoles(
	roles []rbac.EffectiveRole,
	dangerOnly bool,
	nsFilter string,
) []rbac.EffectiveRole {
	if !dangerOnly && nsFilter == "" {
		return roles
	}

	var out []rbac.EffectiveRole
	for _, r := range roles {
		if dangerOnly && !r.Dangerous {
			continue
		}
		if nsFilter != "" {
			// фильтруем по namespace, где действует роль
			if r.SourceNamespace != nsFilter && !r.ClusterScope {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}
