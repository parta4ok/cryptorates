package config

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	toolkitConfig "cryptorates/tools/config"
)

var (
	_ toolkitConfig.Config = (*Config)(nil)
)

type Config struct {
	*koanf.Koanf
}

func NewConfig(path string) *Config {
	k := koanf.New(".")
	err := k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		panic(err)
	}

	return &Config{
		Koanf: k,
	}
}

func (cfg *Config) PublicHTTPPort() string {
	return cfg.String("port.http.public.port")
}

func (cfg *Config) PublicHTTPTimeout() time.Duration {
	return cfg.Duration("port.http.public.timeout")
}

func (cfg *Config) StorageType() string {
	return cfg.String("storage.type")
}

func (cfg *Config) StorageConnstr(storageType string) string {
	return cfg.String(fmt.Sprintf("%s.connection_string", storageType))
}

func (cfg *Config) LogLevel() string {
	return cfg.String("logger.level")
}

func (cfg *Config) AddSource() bool {
	return cfg.Bool("logger.add_source")
}

func (cfg *Config) ServiceName() string {
	return cfg.String("service_name")
}

func (cfg *Config) ServiceVersion() string {
	return cfg.String("version")
}

func (cfg *Config) CronActualizeInterval() string {
	return cfg.String("cron.actualize_rates.interval")
}

func (cfg *Config) JaegerEndpoint() string {
	return cfg.String("tracing.jaeger")
}

func (cfg *Config) IsTracingSwitch() bool {
	return cfg.Bool("tracing.switch_on")
}

func (cfg *Config) MetricsPort() string {
	return cfg.String("metrics.port")
}

func (cfg *Config) MetricsTimeout() time.Duration {
	return cfg.Duration("metrics.timeout")
}

func (cfg *Config) GracefulShutdownTimeout() time.Duration {
	return cfg.Duration("graceful_shutdown.timeout")
}
