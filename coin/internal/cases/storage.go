package cases

import (
	"context"

	"cryptorates/coin/internal/entities"
)

//go:generate mockgen -source=storage.go -destination=./testdata/storage.go -package=testdata
type Storage interface {
	GetAllTitles(ctx context.Context) ([]string, error)
	Store(ctx context.Context, coins []*entities.Coin) error
	GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error)
	GetAggregatedCoins(
		ctx context.Context, titles []string, aggregation string) ([]*entities.Coin, error)
}
