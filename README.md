# RBAC Analyzer for Kubernetes (Go)

## Описание

Утилита анализирует Kubernetes RBAC (Role, ClusterRole, RoleBinding, ClusterRoleBinding)
из YAML-манифестов и строит эффективные права для каждого субъекта:

- User
- Group
- ServiceAccount

По каждому субъекту отображаются:

- Все права (verb + resource + apiGroup + resourceNames)
- Уровень применимости: кластерный / namespace
- Является ли роль потенциально опасной (эвристики)

## Основной сценарий

1. Из кластера выгружаются RBAC-ресурсы:

```bash
kubectl get roles,clusterroles,rolebindings,clusterrolebindings -A -o yaml > rbac.yaml
