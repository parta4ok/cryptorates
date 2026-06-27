package application

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"cryptorates/coin/internal/adapters/config"
	"cryptorates/coin/internal/adapters/provider/baseclient"
	"cryptorates/coin/internal/adapters/provider/coingecco"
	"cryptorates/coin/internal/adapters/storage/pg"
	"cryptorates/coin/internal/cases"
	"cryptorates/coin/internal/ports"
	"cryptorates/coin/internal/ports/http/public"
	baseApp "cryptorates/tools/application"
	"cryptorates/tools/cronjob"
	"cryptorates/tools/port"
)

type App struct {
	cfg *config.Config

	provider    cases.Provider
	storage     cases.Storage
	service     ports.Service
	ratesSource baseclient.RatesSource

	publicPort port.Port
}

func New(cfg *config.Config) *App {
	return &App{
		cfg: cfg,
	}
}

func (app *App) Run() {
	app.initCoinGeccoClient()
	app.initBaseClient()
	app.initStorage()
	app.initService()
	app.initHTTPPublic()

	actualizeCron, err := cronjob.NewJob(
		"actualize rates",
		app.cfg.CronActualizeInterval(),
		app.actualizeRates,
	)
	if err != nil {
		app.fatal(err)
	}

	cron, err := cronjob.NewCron(actualizeCron)
	if err != nil {
		app.fatal(err)
	}

	baseApp := baseApp.NewBuilder(app.cfg).
		WithLogger().
		WithMetricsPort().
		WithTracer().
		AddStarters(app.publicPort.Start, cron.Start).
		AddWaiters(app.publicPort.Stop, cron.Stop).
		Build()

	if err := baseApp.Run(); err != nil {
		app.fatal(err)
	}
}

func (app *App) initCoinGeccoClient() {
	client, err := coingecco.NewClient(os.Getenv("API_KEY"))
	if err != nil {
		app.fatal(err)
	}
	app.ratesSource = client
}

func (app *App) initBaseClient() {
	baseClient, err := baseclient.NewBaseClient(app.ratesSource)
	if err != nil {
		app.fatal(err)
	}

	app.provider = baseClient
}

func (app *App) initStorage() {
	stType := app.cfg.StorageType()
	connString := app.cfg.StorageConnstr(stType)

	st, err := pg.NewStorage(connString)
	if err != nil {
		app.fatal(err)
	}

	app.storage = st
}

func (app *App) initService() {
	service, err := cases.NewService(app.provider, app.storage)
	if err != nil {
		app.fatal(err)
	}

	app.service = service
}

func (app *App) initHTTPPublic() {
	port := app.cfg.PublicHTTPPort()
	timeout := app.cfg.PublicHTTPTimeout()

	srv, err := public.NewServer(app.service, port, timeout)
	if err != nil {
		app.fatal(err)
	}

	app.publicPort = srv
}

func (app *App) actualizeRates() {
	slog.Info("actualize rates started")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := app.service.ActualizeRates(ctx); err != nil {
		slog.Warn("actualize rates failure", "err", err.Error())
	}
}

func (app *App) fatal(err error) {
	log.Fatal(err)
}
