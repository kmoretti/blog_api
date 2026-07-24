package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleasesEmbeddedDefaults(t *testing.T) {
	configPath := t.TempDir()

	err := ensureDefaultConfigFiles(configPath)

	if err != nil {
		t.Fatalf("release embedded defaults: %v", err)
	}
	for _, name := range []string{"system_config.json", "friend_list.json"} {
		path := filepath.Join(configPath, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("stat released default %s: %v", name, err)
		}
	}
}

func TestPreservesExistingConfig(t *testing.T) {
	configPath := t.TempDir()
	path := filepath.Join(configPath, "system_config.json")
	const existing = `{"custom":true}`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatalf("arrange existing config: %v", err)
	}

	err := ensureDefaultConfigFiles(configPath)

	if err != nil {
		t.Fatalf("release embedded defaults: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(content) != existing {
		t.Fatalf("existing config changed to %q", content)
	}
}
