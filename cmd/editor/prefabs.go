package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PrefabInfo holds information about a prefab spec file.
type PrefabInfo struct {
	Name string
	Path string
}

// ListPrefabs scans the prefabs/ folder for YAML files.
func ListPrefabs(dir string) ([]PrefabInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	prefabs := make([]PrefabInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		prefabs = append(prefabs, PrefabInfo{
			Name: name,
			Path: filepath.ToSlash(entry.Name()),
		})
	}
	sort.Slice(prefabs, func(i, j int) bool {
		return prefabs[i].Name < prefabs[j].Name
	})
	return prefabs, nil
}
