package exo

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func getConfig(opts []ConfigOption) Config {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 3000
	}

	config := Config{
		local:        false,
		port:         port,
		etag:         nil,
		logging:      false,
		compress:     false,
		errorHandler: nil,
		cors: CorsConfig{
			enable:           false,
			allowOrigins:     []string{"*"},
			allowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			allowHeaders:     []string{"Origin", "Content-Type", "Accept"},
			allowCredentials: true,
			exposeHeaders:    []string{},
			maxAge:           0,
		},
		auth: AuthConfig{
			enable:                  false,
			hashAlgo:                AuthHashAlgoHMACSHA256,
			refreshHashAlgo:         AuthHashAlgoHMACSHA256,
			secrets:                 []string{"secret"},
			idenityTokenExp:         3600,
			refreshToken:            false,
			refreshTokenExp:         3600,
			refreshTokenRotation:    false,
			refreshTokenAbsoluteExp: 0,
			refreshTokenStealDetect: false,
		},
	}

	for _, opt := range opts {
		opt(&config)
	}

	return config
}

// Config is a struct that holds the configuration for the exo framework.
type Config struct {
	local bool
	port  int
	etag  *bool
	cache *struct {
		Expiration    time.Duration
		ClientControl bool
	}
	logging      bool
	compress     bool
	ssl          *tls.Certificate
	errorHandler func(*fiber.Ctx, error) error
	cors         CorsConfig
	auth         AuthConfig
	db           *gorm.DB
	autoMigrate  bool
}

func (c Config) addr() string {
	if c.local {
		return fmt.Sprintf("localhost:%d", c.port)
	}

	return fmt.Sprintf(":%d", c.port)
}

// ConfigOption is a function that modifies the configuration of the exo framework.
type ConfigOption func(*Config)

// AsLocal sets the local flag in the configuration.
func AsLocal() ConfigOption {
	return func(c *Config) {
		c.local = true
	}
}

// WithPort sets the port in the configuration.
func WithPort(port int) ConfigOption {
	return func(c *Config) {
		c.port = port
	}
}

// WithCors sets the CORS configuration in the configuration.
func WithCors(opts ...CorsOption) ConfigOption {
	return func(c *Config) {
		c.cors = getCorsConfig(&c.cors, opts)
	}
}

// WithAuth sets the authentication configuration in the configuration.
func WithAuth(opts ...AuthOption) ConfigOption {
	return func(c *Config) {
		c.auth = getAuthConfig(&c.auth, opts)
	}
}

// WithETag enables ETag caching using week (true) or strong (false) ETag references.
func WithETag(weak bool) ConfigOption {
	return func(c *Config) {
		c.etag = &weak
	}
}

// WithCache sets the cache configuration in the configuration.
func WithCache(expiration time.Duration, clientControl bool) ConfigOption {
	return func(c *Config) {
		c.cache = &struct {
			Expiration    time.Duration
			ClientControl bool
		}{expiration, clientControl}
	}
}

// WithLogging enables logging in the configuration.
func WithLogging() ConfigOption {
	return func(c *Config) {
		c.logging = true
	}
}

// WithCompress enables compression in the configuration.
func WithCompress() ConfigOption {
	return func(c *Config) {
		c.compress = true
	}
}

// WithSimpleErrorHandler sets the error handler in the configuration. The error handler only logs the error and returns a 500 status code.
func WithSimpleErrorHandler(handler func(error)) ConfigOption {
	return WithFullErrorHandler(func(ctx *fiber.Ctx, err error) error {
		handler(err)

		code := fiber.StatusInternalServerError
		var e *fiber.Error
		if errors.As(err, &e) {
			code = e.Code
		}

		return ctx.SendStatus(code)
	})
}

// WithErrorHandler sets the error handler in the configuration. The error handler is a function that takes the context and the error and returns an error.
func WithFullErrorHandler(handler func(ctx *fiber.Ctx, err error) error) ConfigOption {
	return func(c *Config) {
		c.errorHandler = handler
	}
}

// WithSSL sets the SSL certificate in the configuration.
func WithSSL(cert tls.Certificate) ConfigOption {
	return func(c *Config) {
		c.ssl = &cert
	}
}

// WithDB sets the database connection in the configuration.
func WithDB(db *gorm.DB) ConfigOption {
	return func(c *Config) {
		c.db = db
	}
}

// AutoMigration enables the automatic migration of the database schema in the configuration.
func AutoMigration() ConfigOption {
	return func(c *Config) {
		c.autoMigrate = true
	}
}

// CorsConfig is a struct that holds the configuration for CORS.
type CorsConfig struct {
	enable           bool
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	allowCredentials bool
	exposeHeaders    []string
	maxAge           int
}

// CorsOption is a function that modifies the CORS configuration.
type CorsOption func(*CorsConfig)

// CorsAllowOrigins sets the allowed origins in the CORS configuration.
func CorsAllowOrigins(origins ...string) CorsOption {
	return func(c *CorsConfig) {
		c.allowOrigins = origins
	}
}

// CorsAllowMethods sets the allowed methods in the CORS configuration.
func CorsAllowMethods(methods ...string) CorsOption {
	return func(c *CorsConfig) {
		c.allowMethods = methods
	}
}

// CorsAllowHeaders sets the allowed headers in the CORS configuration.
func CorsAllowHeaders(headers ...string) CorsOption {
	return func(c *CorsConfig) {
		c.allowHeaders = headers
	}
}

// CorsAllowCredentials sets the allow credentials flag in the CORS configuration.
func CorsAllowCredentials(allow bool) CorsOption {
	return func(c *CorsConfig) {
		c.allowCredentials = allow
	}
}

// CorsExposeHeaders sets the exposed headers in the CORS configuration.
func CorsExposeHeaders(headers ...string) CorsOption {
	return func(c *CorsConfig) {
		c.exposeHeaders = headers
	}
}

// CorsMaxAge sets the max age in the CORS configuration.
func CorsMaxAge(age int) CorsOption {
	return func(c *CorsConfig) {
		c.maxAge = age
	}
}

func getCorsConfig(config *CorsConfig, opts []CorsOption) CorsConfig {
	config.enable = true

	for _, opt := range opts {
		opt(config)
	}

	return *config
}

// AuthHashAlgo is a type that represents the hashing algorithm used for authentication.
type AuthHashAlgo string

const (
	// AuthHashAlgoHMACSHA256 represents the HMAC-SHA256 hashing algorithm.
	AuthHashAlgoHMACSHA256 AuthHashAlgo = "HMAC-SHA256"
	// AuthHashAlgoHMACSHA512 represents the HMAC-SHA512 hashing algorithm.
	AuthHashAlgoHMACSHA512 AuthHashAlgo = "HMAC-SHA512"
	// AuthHashAlgoRSA represents the RSA hashing algorithm.
	AuthHashAlgoRSA AuthHashAlgo = "RSA"
	// AuthHashAlgoECDSA represents the ECDSA hashing algorithm.
	AuthHashAlgoECDSA AuthHashAlgo = "ECDSA"
)

// AuthConfig is a struct that holds the configuration for authentication.
type AuthConfig struct {
	enable                  bool
	hashAlgo                AuthHashAlgo
	refreshHashAlgo         AuthHashAlgo
	secrets                 []string // used for the secrets used for the hashing algorithm. For symmetric algorithms, only the first secret is used. For asymmetric algorithms, the first two secrets are used. If refresh token is enabled, the third secret is used for symmetric algorithms and the third and fourth secrets are used for asymmetric algorithms.
	idenityTokenExp         int64    // used for the expiration time of the identity token
	refreshToken            bool     // enable refresh token
	refreshTokenExp         int64    // used for the expiration time of the refresh token. Time in seconds.
	refreshTokenRotation    bool     // enable refresh token rotation
	refreshTokenAbsoluteExp int64    // used for the absolute expiration time of the refresh token. Time in seconds.
	refreshTokenStealDetect bool     // enable steal detection for refresh token
}

// AuthOption is a function that modifies the authentication configuration.
type AuthOption func(*AuthConfig)

// WithAuthHashAlgo sets the hashing algorithm in the authentication configuration.
func WithAuthHashAlgorithms(algos ...AuthHashAlgo) AuthOption {
	return func(c *AuthConfig) {
		if len(algos) == 0 {
			panic("at least one hashing algorithm must be provided")
		}

		c.hashAlgo = algos[0]
		if len(algos) > 1 {
			c.refreshHashAlgo = algos[1]
		} else {
			c.refreshHashAlgo = algos[0]
		}
	}
}

// WithAuthSecrets sets the secrets in the authentication configuration.
func WithAuthSecrets(secrets ...string) AuthOption {
	return func(c *AuthConfig) {
		if len(secrets) == 0 {
			panic("at least one secret must be provided")
		}

		c.secrets = secrets
	}
}

// WithAuthIdentityTokenExp sets the expiration time of the identity token in the authentication configuration. Time in seconds.
func WithAuthIdentityTokenExp(exp int64) AuthOption {
	return func(c *AuthConfig) {
		c.idenityTokenExp = exp
	}
}

// WithAuthRefreshToken enables the refresh token in the authentication configuration.
func WithAuthRefreshToken() AuthOption {
	return func(c *AuthConfig) {
		c.refreshToken = true
	}
}

// WithAuthRefreshTokenExp sets the expiration time of the refresh token in the authentication configuration. Time in seconds.
func WithAuthRefreshTokenExp(exp int64) AuthOption {
	return func(c *AuthConfig) {
		c.refreshTokenExp = exp
	}
}

// WithAuthRefreshTokenRotation enables the refresh token rotation in the authentication configuration.
func WithAuthRefreshTokenRotation() AuthOption {
	return func(c *AuthConfig) {
		c.refreshTokenRotation = true
	}
}

// WithAuthRefreshTokenAbsoluteExp sets the absolute expiration time of the refresh token in the authentication configuration. Time in seconds.
func WithAuthRefreshTokenAbsoluteExp(exp int64) AuthOption {
	return func(c *AuthConfig) {
		c.refreshTokenAbsoluteExp = exp
	}
}

// WithAuthRefreshTokenStealDetect enables the steal detection for the refresh token in the authentication configuration.
func WithAuthRefreshTokenStealDetect() AuthOption {
	return func(c *AuthConfig) {
		c.refreshTokenStealDetect = true
	}
}

func getAuthConfig(config *AuthConfig, opts []AuthOption) AuthConfig {
	config.enable = true

	for _, opt := range opts {
		opt(config)
	}

	return *config
}
