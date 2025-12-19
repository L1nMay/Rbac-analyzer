package loader

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// LoadFromBytes парсит YAML (включая kind: List и multi-doc) напрямую из памяти.
// Это удобно для веба (upload файла).
func LoadFromBytes(content []byte) (*Data, error) {
	data := &Data{}

	docs := splitYAMLDocuments(content)
	for _, doc := range docs {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		parseDocument(doc, data)
	}

	return data, nil
}

// Важно: parseDocument/parseSingleObject/splitYAMLDocuments уже определены в loader.go.
// Мы их переиспользуем.
var _ = yaml.Node{} // чтобы gofmt/линтер не ругался на пустой импорт в некоторых IDE
