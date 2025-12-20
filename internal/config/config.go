package config

type Config struct {
	App      AppConfig      `env-prefix:"APP_"`
	GRPC     GRPCConfig     `env-prefix:"GRPC_"`
	Database DatabaseConfig `env-prefix:"DB_"`
}

type AppConfig struct {
	LogLevel string `env:"LOG_LEVEL" env-default:"info"`
}

type GRPCConfig struct {
	Addr string `env:"ADDR" env-default:":50051"`
}

type DatabaseConfig struct {
	Port     string `env:"PORT" env-default:"5432"`
	Host     string `env:"HOST" env-default:"localhost"`
	Name     string `env:"NAME" env-default:"postgres"`
	User     string `env:"USER" env-default:"user"`
	Password string `env:"PASSWORD"`
}
