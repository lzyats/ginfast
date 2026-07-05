package ymlconfig

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

func TestSaveConfigPreservesCommentsAndEncryptedValues(t *testing.T) {
	sourceFile := filepath.Join("..", "..", "..", "config", "config.yml")
	content, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("read source config failed: %v", err)
	}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(tempFile, content, 0644); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}

	cfg := viper.New()
	cfg.AddConfigPath(tempDir)
	cfg.SetConfigName("config")
	cfg.SetConfigType("yml")
	if err := cfg.ReadInConfig(); err != nil {
		t.Fatalf("read temp config failed: %v", err)
	}

	y := &ymlConfig{
		viper:      cfg,
		mu:         new(sync.RWMutex),
		configPath: tempDir,
		configName: "config",
		configType: "yml",
		pendingSet: map[string]interface{}{
			"system.systemname": "GinFast",
		},
	}

	if err := y.SaveConfig(); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	updated, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("read updated config failed: %v", err)
	}

	updatedText := string(updated)
	if !strings.Contains(updatedText, "jwttokensignkey: \"ENC(") {
		t.Fatalf("encrypted token config was not preserved")
	}
	if !strings.Contains(updatedText, "# 系统LOGO") {
		t.Fatalf("system comment was not preserved")
	}
	if !strings.Contains(updatedText, "# 上传方式") {
		t.Fatalf("upload comment was not preserved")
	}
}
