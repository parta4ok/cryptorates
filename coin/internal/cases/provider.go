package cases

import (
	"context"

	"cryptorates/coin/internal/entities"
)

//go:generate mockgen -source=provider.go -destination=./testdata/provider.go -package=testdata
type Provider interface {
	GetActualRates(ctx context.Context, titles []string) ([]*entities.Coin, error)
}
