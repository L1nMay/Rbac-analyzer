package rbac

import (
	"fmt"
	"strings"
)

// ===== Базовые типы Kubernetes RBAC (упрощённые) =====

type ObjectMeta struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

type PolicyRule struct {
	APIGroups     []string `yaml:"apiGroups" json:"apiGroups"`
	Resources     []string `yaml:"resources" json:"resources"`
	Verbs         []string `yaml:"verbs" json:"verbs"`
	ResourceNames []string `yaml:"resourceNames,omitempty" json:"resourceNames,omitempty"`
}

type Role struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       string       `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata" json:"metadata"`
	Rules      []PolicyRule `yaml:"rules" json:"rules"`
}

type ClusterRole struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       string       `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata" json:"metadata"`
	Rules      []PolicyRule `yaml:"rules" json:"rules"`
}

type RoleRef struct {
	APIGroup string `yaml:"apiGroup" json:"apiGroup"`
	Kind     string `yaml:"kind" json:"kind"`
	Name     string `yaml:"name" json:"name"`
}

type Subject struct {
	Kind      string `yaml:"kind" json:"kind"`
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
}

type RoleBinding struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta `yaml:"metadata" json:"metadata"`
	Subjects   []Subject  `yaml:"subjects" json:"subjects"`
	RoleRef    RoleRef    `yaml:"roleRef" json:"roleRef"`
}

type ClusterRoleBinding struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta `yaml:"metadata" json:"metadata"`
	Subjects   []Subject  `yaml:"subjects" json:"subjects"`
	RoleRef    RoleRef    `yaml:"roleRef" json:"roleRef"`
}

// ===== Наши аналитические типы =====

type SubjectKind string

const (
	SubjectKindUser           SubjectKind = "User"
	SubjectKindGroup          SubjectKind = "Group"
	SubjectKindServiceAccount SubjectKind = "ServiceAccount"
)

type SubjectRef struct {
	Kind      SubjectKind `json:"kind"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
}

func (s SubjectRef) String() string {
	if s.Kind == SubjectKindServiceAccount && s.Namespace != "" {
		return fmt.Sprintf("ServiceAccount:%s/%s", s.Namespace, s.Name)
	}
	return fmt.Sprintf("%s:%s", s.Kind, s.Name)
}

// Permission = нормализованное правило
type Permission struct {
	APIGroup      string   `json:"apiGroup"`
	Resource      string   `json:"resource"`
	Verb          string   `json:"verb"`
	ResourceNames []string `json:"resourceNames,omitempty"`

	Namespace    string `json:"namespace,omitempty"`
	ClusterScope bool   `json:"clusterScope"`
}

type EffectiveRole struct {
	SourceKind      string       `json:"sourceKind"`      // Role / ClusterRole
	SourceName      string       `json:"sourceName"`      // имя роли
	SourceNamespace string       `json:"sourceNamespace"` // для Role
	ClusterScope    bool         `json:"clusterScope"`
	Permissions     []Permission `json:"permissions"`

	Dangerous       bool     `json:"dangerous"`
	DangerReasons   []string `json:"dangerReasons,omitempty"`
	BoundVia        string   `json:"boundVia"`                  // RoleBinding / ClusterRoleBinding
	BindingName     string   `json:"bindingName"`               // имя биндинга
	BindingNS       string   `json:"bindingNS"`                 // namespace биндинга
	BindingSubjects []string `json:"bindingSubjects,omitempty"` // список всех subj в биндинге (для контекста)
}

// SubjectPermissions — итоговая структура:
// ключ = субъект, значение = список его ролей
type SubjectPermissions map[SubjectRef][]EffectiveRole

// ===== Вспомогательные функции =====

func NormalizeVerb(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func subjectRefFromSubject(s Subject, defaultNamespace string) SubjectRef {
	ns := s.Namespace
	if ns == "" && s.Kind == "ServiceAccount" {
		ns = defaultNamespace
	}

	switch s.Kind {
	case "User":
		return SubjectRef{Kind: SubjectKindUser, Name: s.Name}
	case "Group":
		return SubjectRef{Kind: SubjectKindGroup, Name: s.Name}
	case "ServiceAccount":
		return SubjectRef{Kind: SubjectKindServiceAccount, Name: s.Name, Namespace: ns}
	default:
		// неизвестный тип субъекта — отнесём к User по умолчанию
		return SubjectRef{Kind: SubjectKindUser, Name: fmt.Sprintf("%s:%s", s.Kind, s.Name)}
	}
}
