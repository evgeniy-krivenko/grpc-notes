package config

import "time"

type Config struct {
	App         AppConfig      `env-prefix:"APP_"`
	HTTP        HTTPConfig     `env-prefix:"HTTP_"`
	SwaggerHTTP HTTPConfig     `env-prefix:"SWAGGER_HTTP_"`
	GRPC        GRPCConfig     `env-prefix:"GRPC_"`
	Database    DatabaseConfig `env-prefix:"DB_"`
}

type HTTPConfig struct {
	Addr string `env:"ADDR" env-default:":8081"`
}

type AppConfig struct {
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`
	Pretty   bool   `env:"PRETTY" env-default:"false"`
}

type GRPCConfig struct {
	Addr                 string        `env:"ADDR" env-default:":50051"`
	KeepaliveTime        time.Duration `env:"KEEPALIVE_TIME" env-default:"60s"`
	KeepaliveTimeout     time.Duration `env:"KEEPALIVE_TIMEOUT" env-default:"30s"`
	MaxConcurrentStreams uint32        `env:"MAX_CONCURRENT_STREAMS" env-default:"50"`
}

type DatabaseConfig struct {
	Port     string `env:"PORT" env-default:"5432"`
	Host     string `env:"HOST" env-default:"localhost"`
	Name     string `env:"NAME" env-default:"postgres"`
	User     string `env:"USER" env-default:"user"`
	Password string `env:"PASSWORD"`
}
