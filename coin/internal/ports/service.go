package ports

import (
	"context"
	"cryptorates/coin/internal/entities"
)

//go:generate mockgen -source=service.go -destination=./testdata/service.go -package=testdata
type Service interface {
	GetActualRates(ctx context.Context, titles []string) ([]*entities.Coin, error)
	GetAggregateRates(
		ctx context.Context, titles []string, aggregation string) ([]*entities.Coin, error)
	ActualizeRates(ctx context.Context) error
}
