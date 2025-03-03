package internal

import (
	"os"
	"path/filepath"
)

var (
	DefaultAppName          = "perceptionengine"
	DefaultAppCMDShortCut   = "pce"
	DefaultConfigFolderName = DefaultAppName
	DefaultConfigPath       = filepath.Join(os.Getenv("HOME"), ".config", DefaultConfigFolderName)
	DefaultCacheDir         = filepath.Join(DefaultConfigPath, ".cache")
	DefaultCentralDBPath    = filepath.Join(DefaultConfigPath, "central.db")
	DefaultDotDir           = "." + DefaultConfigFolderName
	DefaultConfigFile       = filepath.Join(DefaultDotDir, "config.toml")
	DefaultGlobalConfigFile = filepath.Join(DefaultConfigPath, "config.toml")
)
