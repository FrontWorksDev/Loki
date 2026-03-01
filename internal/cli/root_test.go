package cli

import (
	"testing"
)

func TestExecute_ヘルプ表示(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	t.Cleanup(func() { rootCmd.SetArgs([]string{}) })

	err := Execute()
	if err != nil {
		t.Errorf("Execute() with --help returned error: %v", err)
	}
}

func TestExecute_バージョン表示(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs([]string{}) })

	err := Execute()
	if err != nil {
		t.Errorf("Execute() with --version returned error: %v", err)
	}
}

func TestRootCmd_サブコマンド存在確認(t *testing.T) {
	expected := []string{"compress", "convert"}
	for _, name := range expected {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%sサブコマンドが登録されていません", name)
		}
	}
}
