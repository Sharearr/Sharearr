package sharearr

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	koantf "github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
)

type Config struct {
	Port  int        `koanf:"port"  toml:"port"`
	DB    string     `koanf:"db"    toml:"db"`
	Debug bool       `koanf:"debug" toml:"-"`
	Log   LogConfig  `koanf:"log"   toml:"log"`
	Init  InitConfig `koanf:"init"  toml:"-"`
}

// InitConfig is excluded from the config file — values come from env vars or
// CLI flags only. This keeps credentials out of the config file.
type InitConfig struct {
	User UserConfig `koanf:"user" toml:"-"`
}

type UserConfig struct {
	Email    string `koanf:"email"    toml:"-"`
	Username string `koanf:"username" toml:"-"`
	APIKey   string `koanf:"apikey"   toml:"-"`
}

type LogConfig struct {
	Level string `koanf:"level" toml:"level"`
	File  string `koanf:"file"  toml:"file"`
}

const defaultPort = 8787

var ErrHelp = errors.New("help requested")

func LoadConfig(args []string) (*Config, error) {
	defaultConfigPath := filepath.Join(defaultConfigDir, "sharearr.toml")
	defaultDBPath := filepath.Join(defaultDataDir, "sharearr.db")
	defaultLogPath := filepath.Join(defaultLogDir, "sharearr.log")

	flags := pflag.NewFlagSet("sharearr", pflag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage of sharearr:\n")
		flags.PrintDefaults()
	}
	flags.StringP("config", "c", defaultConfigPath, "Path to config file")
	flags.IntP("port", "p", defaultPort, "HTTP listen port")
	flags.String("db", defaultDBPath, "SQLite DB path")
	flags.Bool("debug", defaultDebugEnabled, "Enable debug")
	flags.String("log-level", defaultLogLevel, "Log level")
	flags.StringP("log-file", "l", defaultLogPath, "Log file path")
	flags.StringP("init-user-email", "e", "", "Email for the initial user")
	flags.StringP("init-user-username", "u", "", "Username for the initial user")
	flags.StringP("init-user-apikey", "k", "", "API key for the initial user")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil, ErrHelp
		}
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	k := koanf.New(".")

	if err := k.Load(confmap.Provider(map[string]any{
		"port":      defaultPort,
		"db":        defaultDBPath,
		"debug":     defaultDebugEnabled,
		"log.level": defaultLogLevel,
		"log.file":  defaultLogPath,
	}, "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	// Config file path: --config flag takes precedence over SHAREARR_CONFIG env var
	configPath, _ := flags.GetString("config")
	if !flags.Changed("config") {
		if v := os.Getenv("SHAREARR_CONFIG"); v != "" {
			configPath = v
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		if err := k.Load(file.Provider(configPath), koantf.Parser()); err != nil {
			return nil, fmt.Errorf("load config file: %w", err)
		}
	}

	envParser := func(s string) string {
		s = strings.TrimPrefix(s, "SHAREARR_")
		s = strings.ToLower(s)
		return strings.ReplaceAll(s, "__", ".")
	}
	if err := k.Load(env.Provider("SHAREARR_", ".", envParser), nil); err != nil {
		return nil, fmt.Errorf("load env vars: %w", err)
	}

	cliParser := func(k string, v string) (string, any) {
		return strings.ReplaceAll(k, "-", "."), v
	}
	if err := k.Load(posflag.ProviderWithValue(flags, ".", k, cliParser), nil); err != nil {
		return nil, fmt.Errorf("load flags: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if _, err := os.Stat(configPath); errors.Is(err, fs.ErrNotExist) {
		if b, err := toml.Marshal(cfg); err == nil {
			os.WriteFile(configPath, b, 0644)
		}
	}

	return &cfg, nil
}
