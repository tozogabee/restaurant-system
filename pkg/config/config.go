// Package config loads service configuration from the environment.
//
// Every service embeds Base and adds its own fields:
//
//	type Config struct {
//	    config.Base
//	    Postgres config.Postgres
//	}
//
//	cfg, err := config.Load[Config]()
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Environment names the deployment target. It controls defaults such as log
// format and whether stack traces are exposed to clients.
type Environment string

const (
	EnvLocal Environment = "local"
	EnvDev   Environment = "dev"
	EnvProd  Environment = "prod"
)

// Base holds the settings every service needs.
type Base struct {
	ServiceName string      `env:"SERVICE_NAME,required"`
	Environment Environment `env:"ENVIRONMENT" envDefault:"local"`
	LogLevel    string      `env:"LOG_LEVEL" envDefault:"info"`

	// HTTPPort is the port the Gin server listens on. Zero disables it.
	HTTPPort int `env:"HTTP_PORT"`
	// GRPCPort is the port the gRPC server listens on. Zero disables it.
	GRPCPort int `env:"GRPC_PORT"`

	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

// IsProd reports whether the service runs in a production-like environment.
func (b Base) IsProd() bool { return b.Environment == EnvProd }

// Postgres describes a connection to a single Postgres database.
type Postgres struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER,required"`
	Password string `env:"POSTGRES_PASSWORD,required"`
	Database string `env:"POSTGRES_DB,required"`
	// Schema is the service-owned schema; it is set as the search_path.
	Schema  string `env:"POSTGRES_SCHEMA,required"`
	SSLMode string `env:"POSTGRES_SSLMODE" envDefault:"disable"`

	MaxConns        int32         `env:"POSTGRES_MAX_CONNS" envDefault:"10"`
	MinConns        int32         `env:"POSTGRES_MIN_CONNS" envDefault:"2"`
	MaxConnLifetime time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"1h"`
	MaxConnIdleTime time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"30m"`
}

// DSN renders the connection string consumed by pgx.
func (p Postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&search_path=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode, p.Schema,
	)
}

// Redis describes a connection to a Redis instance.
type Redis struct {
	Addr     string        `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string        `env:"REDIS_PASSWORD"`
	DB       int           `env:"REDIS_DB" envDefault:"0"`
	Timeout  time.Duration `env:"REDIS_TIMEOUT" envDefault:"3s"`
}

// Load parses T from the process environment.
func Load[T any]() (T, error) {
	var cfg T
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}
