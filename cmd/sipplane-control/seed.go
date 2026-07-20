package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

func readSeedYAML(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return os.ReadFile(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.EqualFold(name, "bootstrap.yaml") || strings.EqualFold(name, "bootstrap.yml") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, name))
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		if !bytes.HasSuffix(data, []byte("\n")) {
			buf.WriteByte('\n')
		}
		buf.WriteString("---\n")
	}
	return buf.Bytes(), nil
}
