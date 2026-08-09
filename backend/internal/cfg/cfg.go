package cfg

import (
	"errors"
	"net"
	"strings"
	"time"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
	defaultHTTPAddress     = ":8080"
)

// Config is the application's complete configuration. It belongs at the
// composition root; lower-level packages should receive only their section.
type Config struct {
	ServiceName   string
	Environment   string `env:"DAFTAR_ENVIRONMENT" envDefault:"development"`
	Logging       Logging
	HTTP          HTTP
	MongoDB       MongoDB
	JWT           JWT
	Cookie        Cookie
	BootstrapUser BootstrapUser
}

type Logging struct {
	Level                string        `env:"DAFTAR_LOG_LEVEL" envDefault:"info"`
	Format               string        `env:"DAFTAR_LOG_FORMAT" envDefault:"pretty"`
	SlowRequestThreshold time.Duration `env:"DAFTAR_LOG_SLOW_REQUEST_THRESHOLD" envDefault:"750ms"`
}

type HTTP struct {
	Address               string        `env:"DAFTAR_HTTP_ADDRESS" envDefault:":8080"`
	ReadHeaderTimeout     time.Duration `env:"DAFTAR_HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout           time.Duration `env:"DAFTAR_HTTP_READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout          time.Duration `env:"DAFTAR_HTTP_WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout           time.Duration `env:"DAFTAR_HTTP_IDLE_TIMEOUT" envDefault:"1m"`
	ShutdownTimeout       time.Duration `env:"DAFTAR_HTTP_SHUTDOWN_TIMEOUT" envDefault:"10s"`
	RequestTimeout        time.Duration `env:"DAFTAR_HTTP_REQUEST_TIMEOUT" envDefault:"1m"`
	MaxHeaderBytes        int           `env:"DAFTAR_HTTP_MAX_HEADER_BYTES" envDefault:"1048576"`
	CORSAllowedOrigins    []string      `env:"DAFTAR_CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`
	RateLimitPerMinute    int           `env:"DAFTAR_RATE_LIMIT_PER_MINUTE" envDefault:"60"`
	LoginRatePerMinute    int           `env:"DAFTAR_LOGIN_RATE_PER_MINUTE" envDefault:"5"`
	RegisterRatePerMinute int           `env:"DAFTAR_REGISTER_RATE_PER_MINUTE" envDefault:"3"`
	TrustedProxies        []string      `env:"DAFTAR_TRUSTED_PROXIES" envSeparator:","`
	EnableHSTS            bool          `env:"DAFTAR_ENABLE_HSTS" envDefault:"false"`
}

type MongoDB struct {
	URI             string        `env:"DAFTAR_MONGODB_URI"`
	Database        string        `env:"DAFTAR_MONGODB_DATABASE" envDefault:"daftar"`
	ConnectTimeout  time.Duration `env:"DAFTAR_MONGODB_CONNECT_TIMEOUT" envDefault:"10s"`
	ShutdownTimeout time.Duration `env:"DAFTAR_MONGODB_SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type JWT struct {
	Secret     string        `env:"DAFTAR_JWT_SECRET"`
	Issuer     string        `env:"DAFTAR_JWT_ISSUER" envDefault:"daftar-api"`
	Audience   string        `env:"DAFTAR_JWT_AUDIENCE" envDefault:"daftar-web"`
	AccessTTL  time.Duration `env:"DAFTAR_JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTTL time.Duration `env:"DAFTAR_JWT_REFRESH_TTL" envDefault:"720h"`
}

type Cookie struct {
	Name        string `env:"DAFTAR_COOKIE_NAME" envDefault:"daftar_session"`
	RefreshName string `env:"DAFTAR_REFRESH_COOKIE_NAME" envDefault:"daftar_refresh"`
	Domain      string `env:"DAFTAR_COOKIE_DOMAIN"`
	Path        string `env:"DAFTAR_COOKIE_PATH" envDefault:"/"`
	Secure      bool   `env:"DAFTAR_COOKIE_SECURE" envDefault:"false"`
	SameSite    string `env:"DAFTAR_COOKIE_SAME_SITE" envDefault:"lax"`
}

type BootstrapUser struct {
	Enabled   bool   `env:"DAFTAR_BOOTSTRAP_USER_ENABLED" envDefault:"false"`
	Email     string `env:"DAFTAR_BOOTSTRAP_USER_EMAIL"`
	Password  string `env:"DAFTAR_BOOTSTRAP_USER_PASSWORD"`
	FirstName string `env:"DAFTAR_BOOTSTRAP_USER_FIRST_NAME"`
	LastName  string `env:"DAFTAR_BOOTSTRAP_USER_LAST_NAME"`
}

func Default() Config {
	return DefaultConfig()
}

func DefaultConfig() Config {
	return Config{
		ServiceName: "daftar-api",
		Environment: EnvironmentDevelopment,
		Logging: Logging{
			Level:                "info",
			Format:               "pretty",
			SlowRequestThreshold: 750 * time.Millisecond,
		},
		HTTP: HTTP{}.WithDefaults(),
		MongoDB: MongoDB{
			ConnectTimeout:  10 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		JWT: JWT{
			Issuer:     "daftar-api",
			Audience:   "daftar-web",
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 30 * 24 * time.Hour,
		},
		Cookie: Cookie{Name: "daftar_session", RefreshName: "daftar_refresh", Path: "/", SameSite: "lax"},
	}
}

func (config Config) Validate() error {
	switch strings.ToLower(config.Logging.Level) {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("cfg: log level must be debug, info, warn, or error")
	}
	switch strings.ToLower(config.Logging.Format) {
	case "pretty", "json":
	default:
		return errors.New("cfg: log format must be pretty or json")
	}
	if config.Logging.SlowRequestThreshold <= 0 {
		return errors.New("cfg: slow request threshold must be positive")
	}
	if len(config.JWT.Secret) < 32 {
		return errors.New("cfg: JWT secret must contain at least 32 characters")
	}
	if strings.TrimSpace(config.JWT.Issuer) == "" || strings.TrimSpace(config.JWT.Audience) == "" || config.JWT.AccessTTL <= 0 || config.JWT.RefreshTTL <= 0 {
		return errors.New("cfg: JWT issuer, audience, and access TTL are required")
	}
	switch strings.ToLower(config.Cookie.SameSite) {
	case "lax", "strict", "none":
	default:
		return errors.New("cfg: cookie SameSite must be lax, strict, or none")
	}
	if strings.EqualFold(config.Cookie.SameSite, "none") && !config.Cookie.Secure {
		return errors.New("cfg: SameSite=None requires a secure cookie")
	}
	if config.BootstrapUser.Enabled {
		bootstrap := config.BootstrapUser
		if strings.TrimSpace(bootstrap.Email) == "" || len(bootstrap.Password) < 8 ||
			strings.TrimSpace(bootstrap.FirstName) == "" || strings.TrimSpace(bootstrap.LastName) == "" {
			return errors.New("cfg: enabled bootstrap user requires email, password of at least 8 characters, first name, and last name")
		}
	}
	for _, proxy := range config.HTTP.TrustedProxies {
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return errors.New("cfg: trusted proxies must contain valid IP addresses or CIDR ranges")
			}
		}
	}
	if config.Environment == EnvironmentProduction {
		if !config.Cookie.Secure {
			return errors.New("cfg: production authentication cookie must be secure")
		}
		for _, origin := range config.HTTP.CORSAllowedOrigins {
			if origin == "*" {
				return errors.New("cfg: production CORS origins cannot contain a wildcard")
			}
		}
	}
	return nil
}

// WithDefaults fills omitted HTTP values without overwriting explicit ones.
func (config HTTP) WithDefaults() HTTP {
	if config.Address == "" {
		config.Address = defaultHTTPAddress
	}
	if config.ReadHeaderTimeout <= 0 {
		config.ReadHeaderTimeout = 5 * time.Second
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 60 * time.Second
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 10 * time.Second
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 60 * time.Second
	}
	if config.MaxHeaderBytes <= 0 {
		config.MaxHeaderBytes = 1 << 20
	}
	if len(config.CORSAllowedOrigins) == 0 {
		config.CORSAllowedOrigins = []string{"http://localhost:3000"}
	}
	if config.RateLimitPerMinute <= 0 {
		config.RateLimitPerMinute = 60
	}
	if config.LoginRatePerMinute <= 0 {
		config.LoginRatePerMinute = 5
	}
	if config.RegisterRatePerMinute <= 0 {
		config.RegisterRatePerMinute = 3
	}

	return config
}
