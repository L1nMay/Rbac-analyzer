package rbac

import (
	"fmt"
	"sort"
)

type DiffResult struct {
	BaseScanID   int64         `json:"baseScanId"`
	TargetScanID int64         `json:"targetScanId"`
	Summary      DiffSummary   `json:"summary"`
	Subjects     []SubjectDiff `json:"subjects"`
}

type DiffSummary struct {
	SubjectsChanged int `json:"subjectsChanged"`
	PermsAdded      int `json:"permsAdded"`
	PermsRemoved    int `json:"permsRemoved"`
	DangerIncreased int `json:"dangerIncreased"`
	DangerDecreased int `json:"dangerDecreased"`
}

type SubjectDiff struct {
	SubjectKey      string   `json:"subjectKey"`
	Added           []string `json:"added"`
	Removed         []string `json:"removed"`
	BaseDangerous   bool     `json:"baseDangerous"`
	TargetDangerous bool     `json:"targetDangerous"`
	BaseReasons     []string `json:"baseReasons"`
	TargetReasons   []string `json:"targetReasons"`
}

func DiffSubjectPermissions(base SubjectPermissions, target SubjectPermissions) DiffResult {
	baseMap := normalizeSubjectPerms(base)
	targetMap := normalizeSubjectPerms(target)

	subjectKeys := make([]string, 0, len(baseMap)+len(targetMap))
	seen := map[string]bool{}

	for k := range baseMap {
		if !seen[k] {
			seen[k] = true
			subjectKeys = append(subjectKeys, k)
		}
	}
	for k := range targetMap {
		if !seen[k] {
			seen[k] = true
			subjectKeys = append(subjectKeys, k)
		}
	}
	sort.Strings(subjectKeys)

	out := DiffResult{
		Summary:  DiffSummary{},
		Subjects: make([]SubjectDiff, 0, len(subjectKeys)),
	}

	for _, sk := range subjectKeys {
		b := baseMap[sk]
		t := targetMap[sk]

		added := make([]string, 0)
		removed := make([]string, 0)

		for pk := range t.permSet {
			if !b.permSet[pk] {
				added = append(added, pk)
			}
		}
		for pk := range b.permSet {
			if !t.permSet[pk] {
				removed = append(removed, pk)
			}
		}

		sort.Strings(added)
		sort.Strings(removed)

		if len(added) == 0 && len(removed) == 0 && b.dangerous == t.dangerous {
			continue
		}

		out.Subjects = append(out.Subjects, SubjectDiff{
			SubjectKey:      sk,
			Added:           added,
			Removed:         removed,
			BaseDangerous:   b.dangerous,
			TargetDangerous: t.dangerous,
			BaseReasons:     b.reasons,
			TargetReasons:   t.reasons,
		})

		out.Summary.SubjectsChanged++
		out.Summary.PermsAdded += len(added)
		out.Summary.PermsRemoved += len(removed)
		if !b.dangerous && t.dangerous {
			out.Summary.DangerIncreased++
		}
		if b.dangerous && !t.dangerous {
			out.Summary.DangerDecreased++
		}
	}

	return out
}

type subjNorm struct {
	permSet   map[string]bool
	dangerous bool
	reasons   []string
}

func normalizeSubjectPerms(sp SubjectPermissions) map[string]subjNorm {
	out := map[string]subjNorm{}

	for k, roles := range sp {
		subjectKey := keyToString(k)

		n := subjNorm{permSet: map[string]bool{}}

		isDanger := false
		reasonsSet := map[string]bool{}

		for _, rp := range roles {
			if rp.Dangerous {
				isDanger = true
			}
			for _, r := range rp.DangerReasons {
				reasonsSet[r] = true
			}
			for _, p := range rp.Permissions {
				key := CanonicalPermissionKey(p.Namespace, p.Verb, p.APIGroup, p.Resource, p.ResourceNames)
				n.permSet[key] = true
			}
		}

		n.dangerous = isDanger
		n.reasons = make([]string, 0, len(reasonsSet))
		for r := range reasonsSet {
			n.reasons = append(n.reasons, r)
		}
		sort.Strings(n.reasons)

		out[subjectKey] = n
	}

	return out
}

// keyToString makes a stable string key from whatever map key type SubjectPermissions uses.
func keyToString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		// Fallback: fmt
		return fmt.Sprintf("%v", x)
	}
}

func CanonicalPermissionKey(ns, verb, apiGroup, resource string, names []string) string {
	scope := "namespace"
	nsVal := ns
	if ns == "" || ns == "*" {
		scope = "cluster"
		if nsVal == "" {
			nsVal = "*"
		}
	}
	if nsVal == "" {
		nsVal = "*"
	}

	namesPart := "[]"
	if len(names) > 0 {
		cp := append([]string{}, names...)
		sort.Strings(cp)
		namesPart = "[" + joinComma(cp) + "]"
	}

	return "scope=" + scope +
		" ns=" + nsVal +
		" verb=" + verb +
		" apiGroup=" + apiGroup +
		" resource=" + resource +
		" names=" + namesPart
}

func joinComma(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	s := xs[0]
	for i := 1; i < len(xs); i++ {
		s += "," + xs[i]
	}
	return s
}
