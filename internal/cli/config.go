package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// cfgFile holds the path to the config file specified by the --config flag.
var cfgFile string

// initConfig initializes Viper configuration.
// Priority: CLI flags > config file > defaults.
func initConfig() {
	// Set defaults (equivalent to configs/default.yaml)
	viper.SetDefault("compress.quality", 0)
	viper.SetDefault("compress.level", "medium")
	viper.SetDefault("compress.output", "")
	viper.SetDefault("compress.recursive", false)

	if cfgFile != "" {
		// Use config file specified by --config flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.SetConfigName(".image-compresser")
			viper.SetConfigType("yaml")
		}
	}

	// Read config file (ignore "not found" error)
	if err := viper.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			// Only report errors other than "not found"
			if cfgFile != "" {
				// If explicitly specified, report the error
				fmt.Fprintf(os.Stderr, "設定ファイルの読み込みに失敗しました: %v\n", err)
			}
		}
	}

	// Bind compress command flags to Viper
	// This is done here (not in init()) so bindings survive viper.Reset() in tests
	bindCompressFlags()
}
