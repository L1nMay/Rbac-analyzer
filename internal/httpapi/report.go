package httpapi

import "rbac-analyzer/internal/rbac"

// ---- Summary helpers (MVP-коммерческий смысл) ----

func BuildSummary(sp rbac.SubjectPermissions) map[string]any {
	type counts struct {
		Subjects    int `json:"subjects"`
		Roles       int `json:"roles"`
		Perms       int `json:"perms"`
		DangerRoles int `json:"dangerRoles"`
	}
	c := counts{}
	topDanger := make([]map[string]any, 0)

	for subj, roles := range sp {
		c.Subjects++
		dCount := 0
		pCount := 0
		for _, r := range roles {
			c.Roles++
			pCount += len(r.Permissions)
			if r.Dangerous {
				c.DangerRoles++
				dCount++
			}
		}
		c.Perms += pCount
		if dCount > 0 {
			topDanger = append(topDanger, map[string]any{
				"subject":     subj.String(),
				"dangerRoles": dCount,
				"perms":       pCount,
			})
		}
	}

	riskScore := 0.0
	if c.Roles > 0 {
		riskScore = float64(c.DangerRoles) / float64(c.Roles) * 10.0
		if riskScore > 10 {
			riskScore = 10
		}
	}

	return map[string]any{
		"counts":    c,
		"riskScore": riskScore,
		"topDanger": topDanger,
	}
}

func BuildFullReport(sp rbac.SubjectPermissions) map[string]any {
	type subjOut struct {
		Subject string               `json:"subject"`
		Roles   []rbac.EffectiveRole `json:"roles"`
	}
	out := make([]subjOut, 0, len(sp))
	for sref, roles := range sp {
		out = append(out, subjOut{
			Subject: sref.String(),
			Roles:   roles,
		})
	}
	return map[string]any{"subjects": out}
}
