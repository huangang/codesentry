package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	LDAP     LDAPConfig     `yaml:"ldap"`
	OpenAI   OpenAIConfig   `yaml:"openai"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release, test
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite, mysql, postgres
	DSN    string `yaml:"dsn"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

type LDAPConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	BaseDN       string `yaml:"base_dn"`
	BindDN       string `yaml:"bind_dn"`
	BindPassword string `yaml:"bind_password"`
	UserFilter   string `yaml:"user_filter"`
	UseSSL       bool   `yaml:"use_ssl"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

var GlobalConfig *Config

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config.yaml"
	}

	var cfg *Config

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg = DefaultConfig()
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return nil, err
		}
		cfg = &fileCfg
	}

	cfg.overrideFromEnv()
	GlobalConfig = cfg
	return cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "8080",
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "codesentry.db",
		},
		JWT: JWTConfig{
			Secret:     "codesentry-secret-key-change-in-production",
			ExpireHour: 24,
		},
		LDAP: LDAPConfig{
			Enabled:    false,
			Port:       389,
			UserFilter: "(uid=%s)",
		},
		OpenAI: OpenAIConfig{
			BaseURL: "https://api.openai.com/v1",
			Model:   "gpt-4",
		},
	}
}

func (c *Config) overrideFromEnv() {
	if host := os.Getenv("SERVER_HOST"); host != "" {
		c.Server.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		c.Server.Port = port
	}
	if mode := os.Getenv("SERVER_MODE"); mode != "" {
		c.Server.Mode = mode
	}
	if driver := os.Getenv("DB_DRIVER"); driver != "" {
		c.Database.Driver = driver
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		c.Database.DSN = dsn
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.JWT.Secret = secret
	}
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		c.OpenAI.BaseURL = baseURL
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		c.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		c.OpenAI.Model = model
	}
}

func (c *Config) Save(configPath string) error {
	if configPath == "" {
		configPath = "config.yaml"
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
