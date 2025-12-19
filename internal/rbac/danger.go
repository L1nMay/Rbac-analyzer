package rbac

import "strings"

// EvaluateDangerous определяет, является ли набор правил "опасным".
func EvaluateDangerous(rules []PolicyRule) (bool, []string) {
	var reasons []string

	for _, rule := range rules {
		verbs := toLowerSet(rule.Verbs)
		resources := toLowerSet(rule.Resources)

		// 1. Полные права "*" на все "*"
		if containsStar(rule.Verbs) && containsStar(rule.Resources) {
			reasons = append(reasons, "Full admin: verbs=* and resources=*")
		}

		// 2. Управление ролями/биндингами => потенциальный privilege escalation
		if hasAny(resources, "roles", "clusterroles", "rolebindings", "clusterrolebindings") &&
			hasAny(verbs, "create", "update", "patch", "delete", "*") {
			reasons = append(reasons, "Can modify RBAC objects (potential privilege escalation)")
		}

		// 3. Работа с secrets
		if hasAny(resources, "secrets", "*") &&
			hasAny(verbs, "get", "list", "watch", "*") {
			reasons = append(reasons, "Can read Secrets (sensitive data exposure)")
		}

		// 4. Exec/attach в поды => удалённое выполнение кода
		if hasAny(resources, "pods/exec", "pods/attach", "pods", "*") &&
			hasAny(verbs, "create", "update", "patch", "delete", "get", "*") {
			reasons = append(reasons, "Can exec/attach into pods (remote code execution)")
		}

		// 5. Управление Pod/Deployment => возможность разворачивать произвольный код
		if hasAny(resources, "pods", "deployments", "statefulsets", "daemonsets", "*") &&
			hasAny(verbs, "create", "update", "patch", "delete", "*") {
			reasons = append(reasons, "Can control workload objects (deploy arbitrary code)")
		}

		// 6. Доступ к ConfigMap (утечка конфигурации)
		if hasAny(resources, "configmaps", "*") &&
			hasAny(verbs, "get", "list", "watch", "*") {
			reasons = append(reasons, "Can read ConfigMaps (configuration/secret leakage)")
		}
	}

	return len(reasons) > 0, uniqueStrings(reasons)
}

func containsStar(items []string) bool {
	for _, v := range items {
		if strings.TrimSpace(v) == "*" {
			return true
		}
	}
	return false
}

func toLowerSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, v := range items {
		m[strings.ToLower(strings.TrimSpace(v))] = struct{}{}
	}
	return m
}

func hasAny(set map[string]struct{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := set[strings.ToLower(k)]; ok {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	m := make(map[string]struct{}, len(in))
	var out []string
	for _, v := range in {
		if _, ok := m[v]; ok {
			continue
		}
		m[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
