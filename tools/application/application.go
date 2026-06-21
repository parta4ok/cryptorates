package application

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptorates/tools/config"
	"cryptorates/tools/logger"
	"cryptorates/tools/metrics"
	"cryptorates/tools/tracing/tracer"
)

type BaseApplication struct {
	waiters         []func(ctx context.Context) error
	starters        []func() error
	shutdownTimeout time.Duration
}

type Builder struct {
	cfg config.Config
	app *BaseApplication
}

func NewBuilder(cfg config.Config) *Builder {
	return &Builder{
		cfg: cfg,
		app: &BaseApplication{},
	}
}

func (b *Builder) WithLogger() *Builder {
	logger.InitLogger(b.cfg)

	return b
}

func (b *Builder) WithMetricsPort() *Builder {
	server, err := metrics.NewServer(b.cfg.MetricsPort(), b.cfg.MetricsTimeout())
	if err != nil {
		panic(err)
	}

	b.app.waiters = append(b.app.waiters, server.Stop)
	b.app.starters = append(b.app.starters, server.Start)
	return b
}

func (b *Builder) WithTracer() *Builder {
	if b.cfg.IsTracingSwitch() {
		tracer.ActivateTracer()
		shutdown, err := tracer.Init(
			context.Background(),
			b.cfg.ServiceName(),
			b.cfg.ServiceVersion(),
			b.cfg.JaegerEndpoint(),
		)
		if err != nil {
			panic(err)
		}
		b.app.waiters = append(b.app.waiters, shutdown)
	}

	return b
}

func (b *Builder) AddStarters(starters ...func() error) *Builder {
	b.app.starters = append(b.app.starters, starters...)

	return b
}

func (b *Builder) AddWaiters(waiters ...func(context.Context) error) *Builder {
	b.app.waiters = append(b.app.waiters, waiters...)

	return b
}

func (b *Builder) Build() *BaseApplication {
	b.app.shutdownTimeout = b.cfg.GracefullShutdownTimeout()
	return b.app
}

func (app *BaseApplication) Start() error {
	for _, startFn := range app.starters {
		if err := startFn(); err != nil {
			return err
		}
	}
	return nil
}

func (app *BaseApplication) Stop(ctx context.Context) error {
	var lastErr error
	for _, stopFn := range app.waiters {
		if err := stopFn(ctx); err != nil {
			slog.Error("shutdown error", "err", err)
			lastErr = err
		}
	}
	return lastErr
}

func (app *BaseApplication) Run() error {
	if err := app.Start(); err != nil {
		return err
	}
	slog.Info("application started")

	osMon := make(chan os.Signal, 1)
	signal.Notify(osMon, syscall.SIGINT, syscall.SIGTERM)
	<-osMon
	slog.Info("shutdown signal received")

	interval := app.shutdownTimeout
	if app.shutdownTimeout == 0 {
		interval = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), interval)
	defer cancel()

	return app.Stop(ctx)
}
