package loader

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"rbac-analyzer/internal/rbac"
)

// Data — собранные RBAC объекты из YAML.
type Data struct {
	Roles               []rbac.Role
	ClusterRoles        []rbac.ClusterRole
	RoleBindings        []rbac.RoleBinding
	ClusterRoleBindings []rbac.ClusterRoleBinding
}

// typeMeta нужен для определения kind
type typeMeta struct {
	Kind string `yaml:"kind"`
}

// listMeta — поддержка kubectl get ... -o yaml (kind: List)
type listMeta struct {
	Items []map[string]interface{} `yaml:"items"`
}

// LoadFromDir рекурсивно читает YAML-файлы и извлекает RBAC-объекты.
func LoadFromDir(root string) (*Data, error) {
	data := &Data{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isYAML(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		docs := splitYAMLDocuments(content)
		for _, doc := range docs {
			if len(bytes.TrimSpace(doc)) == 0 {
				continue
			}
			parseDocument(doc, data)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return data, nil
}

// parseDocument обрабатывает один YAML-документ
func parseDocument(doc []byte, data *Data) {
	var tm typeMeta
	if err := yaml.Unmarshal(doc, &tm); err != nil {
		return
	}

	// kubectl get ... -o yaml
	if tm.Kind == "List" {
		var lm listMeta
		if err := yaml.Unmarshal(doc, &lm); err != nil {
			return
		}
		for _, item := range lm.Items {
			raw, err := yaml.Marshal(item)
			if err != nil {
				continue
			}
			parseSingleObject(raw, data)
		}
		return
	}

	parseSingleObject(doc, data)
}

// parseSingleObject разбирает один RBAC-объект
func parseSingleObject(doc []byte, data *Data) {
	var tm typeMeta
	if err := yaml.Unmarshal(doc, &tm); err != nil {
		return
	}

	switch tm.Kind {
	case "Role":
		var r rbac.Role
		if err := yaml.Unmarshal(doc, &r); err == nil {
			data.Roles = append(data.Roles, r)
		}
	case "ClusterRole":
		var cr rbac.ClusterRole
		if err := yaml.Unmarshal(doc, &cr); err == nil {
			data.ClusterRoles = append(data.ClusterRoles, cr)
		}
	case "RoleBinding":
		var rb rbac.RoleBinding
		if err := yaml.Unmarshal(doc, &rb); err == nil {
			data.RoleBindings = append(data.RoleBindings, rb)
		}
	case "ClusterRoleBinding":
		var crb rbac.ClusterRoleBinding
		if err := yaml.Unmarshal(doc, &crb); err == nil {
			data.ClusterRoleBindings = append(data.ClusterRoleBindings, crb)
		}
	}
}

func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// splitYAMLDocuments делит multi-doc YAML по `---`
func splitYAMLDocuments(content []byte) [][]byte {
	s := string(content)
	parts := strings.Split(s, "\n---")
	out := make([][]byte, 0, len(parts))
	for _, p := range parts {
		out = append(out, []byte(p))
	}
	return out
}
