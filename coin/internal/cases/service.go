package cases

import (
	"context"
	"log/slog"
	"slices"

	"github.com/pkg/errors"

	"cryptorates/coin/internal/entities"
	"cryptorates/tools/tracing/tracer"
)

type Service struct {
	provider Provider
	storage  Storage
}

func NewService(provider Provider, storage Storage) (*Service, error) {
	if provider == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "provider not set")
	}

	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	return &Service{
		provider: provider,
		storage:  storage,
	}, nil
}

func (srv *Service) GetActualRates(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	slog.Info("GetActualRates")
	ctx, span, cancel := tracer.Start(ctx, "GetActualRates cases")
	defer cancel()

	if err := srv.processUnstoredTitles(ctx, titles); err != nil {
		err := errors.Wrap(err, "process unstored titles")
		span.SetError(err)
		slog.Error("processUnstoredTitles", "err", err)
		return nil, err
	}

	coins, err := srv.storage.GetCoins(ctx, titles)
	if err != nil {
		err := errors.Wrap(err, "get coins from storage")
		span.SetError(err)
		slog.Error("GetCoins", "err", err)
		return nil, err
	}

	return coins, nil
}

func (srv *Service) GetAggregateRates(ctx context.Context, titles []string, aggregation string) (
	[]*entities.Coin, error) {
	slog.Info("GetAggregateRates")
	ctx, span, cancel := tracer.Start(ctx, "GetAggregateRates cases")
	defer cancel()

	if err := srv.processUnstoredTitles(ctx, titles); err != nil {
		err := errors.Wrap(err, "process unstored titles")
		span.SetError(err)
		slog.Error("processUnstoredTitles", "err", err)
		return nil, err
	}

	coins, err := srv.storage.GetAggregatedCoins(ctx, titles, aggregation)
	if err != nil {
		err := errors.Wrap(err, "GetAggregatedCoins")
		span.SetError(err)
		slog.Error("GetAggregatedCoins", "err", err)
		return nil, err
	}

	return coins, nil
}

func (srv *Service) processUnstoredTitles(ctx context.Context, titles []string) error {
	slog.Info("processUnstoredTitles")
	ctx, span, cancel := tracer.Start(ctx, "processUnstoredTitles cases")
	defer cancel()

	storedTitles, err := srv.storage.GetAllTitles(ctx)
	if err != nil && !errors.Is(err, entities.ErrNotFound) {
		err := errors.Wrap(err, "get all titles")
		span.SetError(err)
		slog.Error("GetAllTitles", "err", err)
		return err
	}

	unstoredTitles := make([]string, 0)
	for _, requiredTitle := range titles {
		if !slices.Contains(storedTitles, requiredTitle) {
			unstoredTitles = append(unstoredTitles, requiredTitle)
		}
	}

	if len(unstoredTitles) > 0 {
		unstoredCoins, err := srv.provider.GetActualRates(ctx, unstoredTitles)
		if err != nil {
			err := errors.Wrap(err, "get actual rates from provider")
			span.SetError(err)
			slog.Error("GetActualRates", "err", err)
			return err
		}

		if err := srv.storage.Store(ctx, unstoredCoins); err != nil {
			err := errors.Wrap(err, "store unstored coins")
			span.SetError(err)
			slog.Error("Store", "err", err)
			return err
		}
	}

	return nil
}

func (srv *Service) ActualizeRates(ctx context.Context) error {
	slog.Info("ActualizeRates")
	ctx, span, cancel := tracer.Start(ctx, "ActualizeRates cases")
	defer cancel()

	allTitles, err := srv.storage.GetAllTitles(ctx)
	if err != nil {
		err := errors.Wrap(err, "get all titles")
		span.SetError(err)
		slog.Error("GetAllTitles", "err", err)
		return err
	}

	renewCoins, err := srv.provider.GetActualRates(ctx, allTitles)
	if err != nil {
		err := errors.Wrap(err, "get actual rates from provider")
		span.SetError(err)
		slog.Error("GetActualRates", "err", err)
		return err
	}

	if err := srv.storage.Store(ctx, renewCoins); err != nil {
		err := errors.Wrap(err, "store renew coins to storage")
		span.SetError(err)
		slog.Error("Store", "err", err)
		return err
	}

	return nil
}
