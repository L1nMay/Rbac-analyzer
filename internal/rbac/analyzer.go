package rbac

// BuildSubjectPermissions строит итоговую карту "субъект -> список ролей".
func BuildSubjectPermissions(
	roles []Role,
	clusterRoles []ClusterRole,
	roleBindings []RoleBinding,
	clusterRoleBindings []ClusterRoleBinding,
) SubjectPermissions {
	result := make(SubjectPermissions)

	roleIndex := indexRoles(roles)
	clusterRoleIndex := indexClusterRoles(clusterRoles)

	// Проходим по RoleBinding
	for _, rb := range roleBindings {
		sourceNamespace := rb.Metadata.Namespace
		allSubjects := collectSubjectsStrings(rb.Subjects, sourceNamespace)

		for _, subj := range rb.Subjects {
			sref := subjectRefFromSubject(subj, sourceNamespace)
			if sref.Name == "" {
				continue
			}

			switch rb.RoleRef.Kind {
			case "Role":
				if role, ok := roleIndex[roleKey(sourceNamespace, rb.RoleRef.Name)]; ok {
					eff := buildEffectiveRoleFromRole(role, rb, allSubjects)
					result[sref] = append(result[sref], eff)
				}
			case "ClusterRole":
				if cr, ok := clusterRoleIndex[rb.RoleRef.Name]; ok {
					eff := buildEffectiveRoleFromClusterRole(cr, rb, allSubjects, false)
					result[sref] = append(result[sref], eff)
				}
			default:
				// неизвестный тип RoleRef — игнорируем
			}
		}
	}

	// Проходим по ClusterRoleBinding
	for _, crb := range clusterRoleBindings {
		allSubjects := collectSubjectsStrings(crb.Subjects, "")
		for _, subj := range crb.Subjects {
			sref := subjectRefFromSubject(subj, "")
			if sref.Name == "" {
				continue
			}

			if cr, ok := clusterRoleIndex[crb.RoleRef.Name]; ok {
				eff := buildEffectiveRoleFromClusterRole(cr, crb, allSubjects, true)
				result[sref] = append(result[sref], eff)
			}
		}
	}

	return result
}

func indexRoles(roles []Role) map[string]*Role {
	m := make(map[string]*Role, len(roles))
	for i := range roles {
		r := &roles[i]
		key := roleKey(r.Metadata.Namespace, r.Metadata.Name)
		m[key] = r
	}
	return m
}

func roleKey(ns, name string) string {
	return ns + "/" + name
}

func indexClusterRoles(clusterRoles []ClusterRole) map[string]*ClusterRole {
	m := make(map[string]*ClusterRole, len(clusterRoles))
	for i := range clusterRoles {
		cr := &clusterRoles[i]
		m[cr.Metadata.Name] = cr
	}
	return m
}

func collectSubjectsStrings(subjects []Subject, defaultNS string) []string {
	out := make([]string, 0, len(subjects))
	for _, s := range subjects {
		ref := subjectRefFromSubject(s, defaultNS)
		out = append(out, ref.String())
	}
	return out
}

func buildEffectiveRoleFromRole(role *Role, rb RoleBinding, allSubjects []string) EffectiveRole {
	perms := flattenRules(role.Rules, role.Metadata.Namespace, false)
	dangerous, reasons := EvaluateDangerous(role.Rules)

	return EffectiveRole{
		SourceKind:      "Role",
		SourceName:      role.Metadata.Name,
		SourceNamespace: role.Metadata.Namespace,
		ClusterScope:    false,
		Permissions:     perms,
		Dangerous:       dangerous,
		DangerReasons:   reasons,
		BoundVia:        "RoleBinding",
		BindingName:     rb.Metadata.Name,
		BindingNS:       rb.Metadata.Namespace,
		BindingSubjects: allSubjects,
	}
}

type bindingLike interface {
	GetKind() string
	GetMetadata() ObjectMeta
}

func buildEffectiveRoleFromClusterRole(
	cr *ClusterRole,
	binding interface{},
	allSubjects []string,
	clusterScope bool,
) EffectiveRole {
	perms := flattenRules(cr.Rules, "", clusterScope)
	dangerous, reasons := EvaluateDangerous(cr.Rules)

	var boundVia, bindingName, bindingNS string

	switch b := binding.(type) {
	case RoleBinding:
		boundVia = "RoleBinding"
		bindingName = b.Metadata.Name
		bindingNS = b.Metadata.Namespace
	case ClusterRoleBinding:
		boundVia = "ClusterRoleBinding"
		bindingName = b.Metadata.Name
		bindingNS = "" // cluster-wide
	default:
		boundVia = "UnknownBinding"
	}

	return EffectiveRole{
		SourceKind:      "ClusterRole",
		SourceName:      cr.Metadata.Name,
		SourceNamespace: "",
		ClusterScope:    clusterScope,
		Permissions:     perms,
		Dangerous:       dangerous,
		DangerReasons:   reasons,
		BoundVia:        boundVia,
		BindingName:     bindingName,
		BindingNS:       bindingNS,
		BindingSubjects: allSubjects,
	}
}

func flattenRules(rules []PolicyRule, namespace string, clusterScope bool) []Permission {
	var perms []Permission
	for _, r := range rules {
		apiGroups := r.APIGroups
		if len(apiGroups) == 0 {
			apiGroups = []string{""}
		}
		resources := r.Resources
		if len(resources) == 0 {
			resources = []string{""}
		}
		verbs := r.Verbs
		if len(verbs) == 0 {
			verbs = []string{""}
		}

		for _, g := range apiGroups {
			for _, res := range resources {
				for _, verb := range verbs {
					perms = append(perms, Permission{
						APIGroup:      g,
						Resource:      res,
						Verb:          verb,
						ResourceNames: r.ResourceNames,
						Namespace:     namespace,
						ClusterScope:  clusterScope,
					})
				}
			}
		}
	}
	return perms
}
