package cli

import (
	"testing"
)

func TestExecute_ヘルプ表示(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	defer rootCmd.SetArgs(nil)

	err := Execute()
	if err != nil {
		t.Errorf("Execute() with --help returned error: %v", err)
	}
}

func TestExecute_バージョン表示(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})
	defer rootCmd.SetArgs(nil)

	err := Execute()
	if err != nil {
		t.Errorf("Execute() with --version returned error: %v", err)
	}
}

func TestRootCmd_サブコマンド存在確認(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "compress" {
			found = true
			break
		}
	}
	if !found {
		t.Error("compressサブコマンドが登録されていません")
	}
}
