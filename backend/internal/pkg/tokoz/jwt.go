package tokoz

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrNotConfigured = errors.New("tokoz: JWT is not configured")
	ErrInvalidConfig = errors.New("tokoz: invalid JWT configuration")
)

type CustomClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type Config struct {
	Secret    string
	Issuer    string
	Audience  string
	AccessTTL time.Duration
}

type tokenConfig struct {
	secret    []byte
	issuer    string
	audience  string
	accessTTL time.Duration
}

var (
	cfgMu  sync.RWMutex
	cfgVal *tokenConfig
)

func Configure(config Config) error {
	if len(config.Secret) < 32 || strings.TrimSpace(config.Issuer) == "" ||
		strings.TrimSpace(config.Audience) == "" || config.AccessTTL <= 0 {
		return ErrInvalidConfig
	}

	cfgMu.Lock()
	cfgVal = &tokenConfig{
		secret:    []byte(config.Secret),
		issuer:    config.Issuer,
		audience:  config.Audience,
		accessTTL: config.AccessTTL,
	}
	cfgMu.Unlock()
	return nil
}

func getConfig() (tokenConfig, error) {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	if cfgVal == nil {
		return tokenConfig{}, ErrNotConfigured
	}
	return tokenConfig{
		secret: append([]byte(nil), cfgVal.secret...), issuer: cfgVal.issuer,
		audience: cfgVal.audience, accessTTL: cfgVal.accessTTL,
	}, nil
}

func GenerateToken(userID, role, email string) (string, error) {
	config, err := getConfig()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	claims := CustomClaims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: config.issuer, Subject: userID, Audience: []string{config.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(config.accessTTL)),
			NotBefore: jwt.NewNumericDate(now), IssuedAt: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(config.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func ValidateAccessToken(tokenString string) (*CustomClaims, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwt.Token) (any, error) { return config.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(config.issuer),
		jwt.WithAudience(config.audience),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}
