package config

import "time"

type Config interface {
	ServiceName() string
	ServiceVersion() string
	LogLevel() string
	AddSource() bool
	IsTracingSwitch() bool
	JaegerEndpoint() string
	MetricsPort() string
	MetricsTimeout() time.Duration
	GracefullShutdownTimeout() time.Duration
}
