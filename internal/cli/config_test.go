package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// resetFlagsAndViper resets viper and pflag Changed states for direct initConfig() tests.
func resetFlagsAndViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	cfgFile = ""
	for _, name := range []string{"quality", "level", "output", "recursive"} {
		if f := compressCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
	for _, name := range []string{"format", "quality", "level", "output", "recursive"} {
		if f := convertCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
	if f := rootCmd.PersistentFlags().Lookup("config"); f != nil {
		f.Changed = false
	}
}

func TestInitConfig_デフォルト値(t *testing.T) {
	resetFlagsAndViper(t)

	initConfig()

	if got := viper.GetInt("compress.quality"); got != 0 {
		t.Errorf("compress.quality = %d, want 0", got)
	}
	if got := viper.GetString("compress.level"); got != "medium" {
		t.Errorf("compress.level = %q, want %q", got, "medium")
	}
	if got := viper.GetString("compress.output"); got != "" {
		t.Errorf("compress.output = %q, want %q", got, "")
	}
	if got := viper.GetBool("compress.recursive"); got != false {
		t.Errorf("compress.recursive = %v, want false", got)
	}

	// convert defaults
	if got := viper.GetString("convert.format"); got != "" {
		t.Errorf("convert.format = %q, want %q", got, "")
	}
	if got := viper.GetInt("convert.quality"); got != 0 {
		t.Errorf("convert.quality = %d, want 0", got)
	}
	if got := viper.GetString("convert.level"); got != "medium" {
		t.Errorf("convert.level = %q, want %q", got, "medium")
	}
	if got := viper.GetString("convert.output"); got != "" {
		t.Errorf("convert.output = %q, want %q", got, "")
	}
	if got := viper.GetBool("convert.recursive"); got != false {
		t.Errorf("convert.recursive = %v, want false", got)
	}
}

func TestInitConfig_設定ファイル読み込み(t *testing.T) {
	resetFlagsAndViper(t)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test-config.yaml")

	content := []byte(`compress:
  quality: 80
  level: "high"
  output: "/tmp/output"
  recursive: true
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfgFile = cfgPath
	t.Cleanup(func() {
		cfgFile = ""
	})

	initConfig()

	if got := viper.GetInt("compress.quality"); got != 80 {
		t.Errorf("compress.quality = %d, want 80", got)
	}
	if got := viper.GetString("compress.level"); got != "high" {
		t.Errorf("compress.level = %q, want %q", got, "high")
	}
	if got := viper.GetString("compress.output"); got != "/tmp/output" {
		t.Errorf("compress.output = %q, want %q", got, "/tmp/output")
	}
	if got := viper.GetBool("compress.recursive"); got != true {
		t.Errorf("compress.recursive = %v, want true", got)
	}
}

func TestInitConfig_設定ファイルなし(t *testing.T) {
	resetFlagsAndViper(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	initConfig()

	if got := viper.GetString("compress.level"); got != "medium" {
		t.Errorf("compress.level = %q, want %q", got, "medium")
	}
}

func TestConfig_フラグ優先(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test-config.yaml")
	content := []byte(`compress:
  quality: 80
  level: "high"
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 90)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.jpg")

	// CLI flag --quality 50 should override config file's quality=80
	rootCmd.SetArgs([]string{"compress", inputPath, "--config", cfgPath, "--quality", "50", "-o", outputPath})
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := viper.GetInt("compress.quality"); got != 50 {
		t.Errorf("compress.quality = %d, want 50 (flag should override config)", got)
	}
}

func TestConfig_設定ファイル優先(t *testing.T) {
	resetGlobals(t)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test-config.yaml")
	content := []byte(`compress:
  level: "high"
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	inputPath := filepath.Join(tmpDir, "test.jpg")
	jpegData := createTestJPEG(t, 10, 10, 90)
	if err := os.WriteFile(inputPath, jpegData, 0o644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "output.jpg")

	// No --level flag, so config file's level=high should override default "medium"
	rootCmd.SetArgs([]string{"compress", inputPath, "--config", cfgPath, "-o", outputPath})
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got := viper.GetString("compress.level"); got != "high" {
		t.Errorf("compress.level = %q, want %q (config should override default)", got, "high")
	}
}
