package pg

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"cryptorates/coin/internal/cases"
	"cryptorates/coin/internal/entities"
	"cryptorates/tools/tracing/tracer"
)

const (
	AggregationMin = "MIN"
	AggregationMax = "MAX"
	AggregationAvg = "AVG"
)

var _ cases.Storage = (*Storage)(nil)

type Storage struct {
	db       *pgxpool.Pool
	cancelFn context.CancelFunc
	once     sync.Once
}

func NewStorage(connString string) (*Storage, error) {
	if connString == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "connection string not set")
	}

	ctx, cancel := context.WithCancel(context.Background())

	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		cancel()
		return nil, errors.Wrapf(entities.ErrInternal, "new pool err: %v", err)
	}

	return &Storage{
		db:       db,
		cancelFn: cancel,
	}, nil
}

func (storage *Storage) Close() {
	storage.once.Do(func() {
		storage.cancelFn()
	})
}

func (storage *Storage) GetAllTitles(ctx context.Context) ([]string, error) {
	slog.Info("GetAllTitles")
	ctx, span, cancel := tracer.Start(ctx, "GetAllTitles adapter storage")
	defer cancel()

	query := `SELECT DISTINCT title FROM crypto.coins`

	rows, err := storage.db.Query(ctx, query)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "get all titles: %v", err)
		span.SetError(err)
		slog.Error("Query", "err", err)
		return nil, err
	}

	defer rows.Close()

	allTitles := make([]string, 0)

	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan title: %v", err)
			span.SetError(err)
			slog.Error("Scan", "err", err)
			return nil, err
		}
		allTitles = append(allTitles, title)
	}

	if rows.Err() != nil {
		err := errors.Wrapf(entities.ErrInternal, "rows has err: %v", rows.Err())
		span.SetError(err)
		slog.Error("rows.Err", "err", err)
		return nil, err
	}

	if len(allTitles) == 0 {
		err := errors.Wrap(entities.ErrNotFound, "not found required titles")
		span.SetError(err)
		slog.Error("Scan", "err", err)
		return nil, err
	}

	return allTitles, nil
}

func (storage *Storage) Store(ctx context.Context, coins []*entities.Coin) error {
	slog.Info("Store", "count", len(coins))
	ctx, span, cancel := tracer.Start(ctx, "Store adapter storage")
	defer cancel()

	query := `INSERT INTO crypto.coins (title, cost, actual_at) values ($1, $2, $3)`

	batch := &pgx.Batch{}
	for _, coin := range coins {
		batch.Queue(
			query,
			coin.Title(),
			coin.Cost(),
			coin.ActualAt(),
		)
	}

	batchRes := storage.db.SendBatch(ctx, batch)

	for range coins {
		if _, err := batchRes.Exec(); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "batch result exec err: %v", err)
			span.SetError(err)
			slog.Error("Exec", "err", err)
			return err
		}
	}

	if err := batchRes.Close(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "batch result close err: %v", err)
		span.SetError(err)
		slog.Error("Close", "err", err)
		return err
	}

	return nil
}

func (storage *Storage) GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	slog.Info("GetCoins", "titles", titles)
	ctx, _, cancel := tracer.Start(ctx, "GetCoins adapter storage")
	defer cancel()

	query := `SELECT DISTINCT ON (c.title) c.title, c.cost, c.actual_at
				FROM crypto.coins c
				WHERE c.title = ANY($1)
				ORDER BY c.title, c.actual_at DESC;`
	params := []any{titles}

	return storage.processRows(ctx, query, params)
}

func (storage *Storage) processRows(
	ctx context.Context, query string, params []any) ([]*entities.Coin, error) {
	ctx, span, cancel := tracer.Start(ctx, "processRows adapter storage")
	defer cancel()

	rows, err := storage.db.Query(ctx, query, params...)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "query err: %v", err)
		span.SetError(err)
		slog.Error("Query", "err", err)
		return nil, err
	}

	coins := make([]*entities.Coin, 0)
	for rows.Next() {
		var (
			title    string
			cost     float64
			actualAt time.Time
		)

		if err := rows.Scan(&title, &cost, &actualAt); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "scan err: %v", err)
			span.SetError(err)
			slog.Error("Scan", "err", err)
			return nil, err
		}

		coin, err := entities.NewCoin(title, cost, entities.WithConcreteActualAt(actualAt))
		if err != nil {
			err := errors.Wrap(err, "new coin from db")
			span.SetError(err)
			slog.Error("NewCoin", "err", err)
			return nil, err
		}

		coins = append(coins, coin)
	}

	if len(coins) == 0 {
		err := errors.Wrapf(entities.ErrNotFound, "titles not found: %v", params...)
		span.SetError(err)
		slog.Error("processRows", "err", err)
		return nil, err
	}

	return coins, nil
}

func (storage *Storage) GetAggregatedCoins(
	ctx context.Context, titles []string, aggregation string) ([]*entities.Coin, error) {
	slog.Info("GetAggregatedCoins", "titles", titles, "aggregation", aggregation)
	ctx, span, cancel := tracer.Start(ctx, "GetAggregatedCoins adapter storage")
	defer cancel()

	var aggFunc string
	switch aggregation {
	case AggregationMin, AggregationMax, AggregationAvg:
		aggFunc = aggregation
	default:
		err := errors.Wrapf(entities.ErrInvalidParam, "invalid aggregate func: %s", aggregation)
		span.SetError(err)
		slog.Error("GetAggregatedCoins", "err", err)
		return nil, err
	}

	query := `
        SELECT c.title, ` + aggFunc + `(c.cost) as cost, CURRENT_DATE as actual_at
        FROM crypto.coins c
        WHERE c.title = ANY($1)
          AND DATE(c.actual_at) = CURRENT_DATE
        GROUP BY c.title
        ORDER BY c.title
    `

	return storage.processRows(ctx, query, []any{titles})
}
