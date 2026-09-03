package toolruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

type catalogEntry struct {
	Name       string          `json:"name"`
	Missing    bool            `json:"missing,omitempty"`
	Descriptor *ToolDescriptor `json:"descriptor,omitempty"`
}

func (runtime *Runtime) CatalogHash(names []string) string {
	clean := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			seen[name] = true
			clean = append(clean, name)
		}
	}
	sort.Strings(clean)
	entries := make([]catalogEntry, 0, len(clean))
	for _, name := range clean {
		descriptor, ok := runtime.Descriptor(name)
		if !ok {
			entries = append(entries, catalogEntry{Name: name, Missing: true})
			continue
		}
		copy := descriptor
		entries = append(entries, catalogEntry{Name: name, Descriptor: &copy})
	}
	raw, _ := json.Marshal(entries)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
