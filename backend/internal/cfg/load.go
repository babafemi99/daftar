package cfg

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// Load reads an optional dotenv file, overlays process environment variables,
// applies defaults, and validates the result. A missing dotenv file is not an
// error because production configuration normally comes from the environment.
func Load(paths ...string) (Config, error) {
	path := ".env"
	if len(paths) > 0 && strings.TrimSpace(paths[0]) != "" {
		path = paths[0]
	}
	_ = godotenv.Load(path)

	config := DefaultConfig()
	if err := env.Parse(&config); err != nil {
		return Config{}, fmt.Errorf("cfg: parse environment: %w", err)
	}
	config.ServiceName = "daftar-api"
	config.HTTP = config.HTTP.WithDefaults()
	if config.Environment == EnvironmentProduction && !envWasSet("DAFTAR_LOG_FORMAT") {
		config.Logging.Format = "json"
	}
	if config.Environment == EnvironmentProduction && !envWasSet("DAFTAR_COOKIE_SECURE") {
		config.Cookie.Secure = true
	}
	if config.Environment == EnvironmentProduction && !envWasSet("DAFTAR_ENABLE_HSTS") {
		config.HTTP.EnableHSTS = true
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// New is the error-returning constructor used by application bootstraps.
func New(paths ...string) (Config, error) {
	return Load(paths...)
}

// Must loads configuration and panics when startup configuration is invalid.
// Prefer Load in main when graceful logging and exit handling are desired.
func Must(paths ...string) Config {
	config, err := Load(paths...)
	if err != nil {
		panic(err)
	}
	return config
}

func envWasSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}
