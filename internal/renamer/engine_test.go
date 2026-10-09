package renamer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnicodeNormalize(t *testing.T) {
	styled := "𝐘𝐮𝐤𝐤𝐢 ʏᴜᴋᴋɪ 𝒀𝒖𝒌𝒌𝒊 𝐒υᴘᴘσꝛᴛ"
	normalized := NormalizeToASCII(styled)

	if !strings.Contains(normalized, "Yukki") {
		t.Errorf("Expected 'Yukki' in normalized string, got: %s", normalized)
	}
	if !strings.Contains(normalized, "Support") {
		t.Errorf("Expected 'Support' in normalized string, got: %s", normalized)
	}
}

func TestStylizeFunctions(t *testing.T) {
	text := "Support"
	aesthetic := ToAestheticFancy(text)
	if !strings.Contains(aesthetic, "𝐒") {
		t.Errorf("Expected stylized '𝐒' in aesthetic text, got: %s", aesthetic)
	}

	bold := ToBoldSerif("Help")
	if bold != "𝐇𝐞𝐥𝐩" {
		t.Errorf("Expected '𝐇𝐞𝐥𝐩', got: %s", bold)
	}
}

func TestEngineRenaming(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "renamer_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "yukki_core")
	_ = os.MkdirAll(subDir, 0755)

	testFile1 := filepath.Join(subDir, "yukki_player.py")
	content1 := `class YukkiPlayer:
    def __init__(self):
        self.name = "YUKKIMUSIC"
        self.repo = "yukki/player"
`
	_ = os.WriteFile(testFile1, []byte(content1), 0644)

	testFile2 := filepath.Join(tempDir, "README.md")
	content2 := "# ʏᴜᴋᴋɪ ᴍᴜꜱɪᴄ\nWelcome to Yukki project!"
	_ = os.WriteFile(testFile2, []byte(content2), 0644)

	engine := NewEngine()
	report, err := engine.Execute(RenameOptions{
		TargetDir:    tempDir,
		OldName:      "Yukki",
		NewName:      "Pulse",
		IncludeFonts: true,
	})

	if err != nil {
		t.Fatalf("Renaming execution failed: %v", err)
	}

	if report.FilesModified < 1 {
		t.Errorf("Expected at least 1 modified file, got: %d", report.FilesModified)
	}

	newSubDir := filepath.Join(tempDir, "pulse_core")
	if _, err := os.Stat(newSubDir); os.IsNotExist(err) {
		t.Errorf("Expected folder 'pulse_core' to exist, but not found")
	}

	newFile1 := filepath.Join(newSubDir, "pulse_player.py")
	data1, err := os.ReadFile(newFile1)
	if err != nil {
		t.Fatalf("Failed to read renamed file: %v", err)
	}

	contentStr := string(data1)
	if !strings.Contains(contentStr, "PulsePlayer") {
		t.Errorf("Expected 'PulsePlayer' in renamed file, got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "PULSEMUSIC") {
		t.Errorf("Expected 'PULSEMUSIC' in renamed file, got: %s", contentStr)
	}
}

func TestFindOccurrencesAndSyntax(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "finder_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	authorFile := filepath.Join(tempDir, "author.txt")
	_ = os.WriteFile(authorFile, []byte("Developed by rahul and 𝐑ᴀʜυʟ"), 0644)

	jsonFile := filepath.Join(tempDir, "config.json")
	_ = os.WriteFile(jsonFile, []byte(`{"maintainer": "rahul"}`), 0644)

	engine := NewEngine()
	findRep, err := engine.FindOccurrences(FindOptions{
		TargetDir:    tempDir,
		SearchTerm:   "rahul",
		IncludeFonts: true,
	})
	if err != nil {
		t.Fatalf("FindOccurrences error: %v", err)
	}

	if findRep.TotalHits < 2 {
		t.Errorf("Expected at least 2 hits for rahul, got: %d", findRep.TotalHits)
	}

	syntaxRes := VerifyDirectorySyntax(tempDir)
	if len(syntaxRes.Warnings) != 0 {
		t.Errorf("Expected 0 syntax warnings, got: %v", syntaxRes.Warnings)
	}
}
