package renamer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModuleNameFromAppJSON(t *testing.T) {
	tempDir := t.TempDir()
	appJsonContent := `{"name": "YukkiMusic", "description": "Music bot"}`
	_ = os.WriteFile(filepath.Join(tempDir, "app.json"), []byte(appJsonContent), 0644)

	detected := DetectModuleName(tempDir, "")
	if detected != "Yukki" {
		t.Fatalf("expected 'Yukki', got '%s'", detected)
	}
}

func TestDetectModuleNameFromPyproject(t *testing.T) {
	tempDir := t.TempDir()
	pyprojectContent := `[project]
name = "PulseMusic"
version = "1.0.0"
`
	_ = os.WriteFile(filepath.Join(tempDir, "pyproject.toml"), []byte(pyprojectContent), 0644)

	detected := DetectModuleName(tempDir, "")
	if detected != "Pulse" {
		t.Fatalf("expected 'Pulse', got '%s'", detected)
	}
}

func TestDetectModuleNameFromGoMod(t *testing.T) {
	tempDir := t.TempDir()
	goModContent := `module github.com/user/MyCoolService

go 1.25.0
`
	_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)

	detected := DetectModuleName(tempDir, "")
	if detected != "MyCoolService" {
		t.Fatalf("expected 'MyCoolService', got '%s'", detected)
	}
}

func TestDetectModuleNameFromGoModIgnoringMain(t *testing.T) {
	tempDir := t.TempDir()
	goModContent := `module main

go 1.25.0
`
	_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644)
	appJsonContent := `{"name": "AnonXMusic"}`
	_ = os.WriteFile(filepath.Join(tempDir, "app.json"), []byte(appJsonContent), 0644)

	detected := DetectModuleName(tempDir, "")
	if detected != "AnonX" {
		t.Fatalf("expected 'AnonX', got '%s'", detected)
	}
}

func TestDetectModuleNameFromRepoURL(t *testing.T) {
	tempDir := t.TempDir()
	url := "https://github.com/group-66666/YukkiMusic-Go.git"
	detected := DetectModuleName(tempDir, url)
	if detected != "Yukki" {
		t.Fatalf("expected 'Yukki', got '%s'", detected)
	}
}

func TestGenerateReplacementPairsDeduplication(t *testing.T) {
	pairs := generateReplacementPairs("Yukki", "Pulse")
	for _, p := range pairs {
		if p.Old == "YUKKIMUSIC" && p.New != "PULSEMUSIC" {
			t.Fatalf("expected PULSEMUSIC, got %s", p.New)
		}
	}

	pairsWithMusic := generateReplacementPairs("Yukki", "PulseMusic")
	for _, p := range pairsWithMusic {
		if p.Old == "YUKKIMUSIC" && p.New != "PULSEMUSIC" {
			t.Fatalf("expected PULSEMUSIC (no duplicate suffix), got %s", p.New)
		}
	}
}
