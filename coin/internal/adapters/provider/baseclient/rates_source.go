package baseclient

import (
	"context"
	"io"
	"net/http"

	"cryptorates/coin/internal/entities"
)

//go:generate mockgen -source=rates_source.go -destination=./testdata/rates_source.go -package=testdata
type RatesSource interface {
	GetRequestData(
		ctx context.Context, titles []string, costIn string) (*http.Request, error)
	ParseResponseData(ctx context.Context, body io.Reader, costIn string) ([]*entities.Coin, error)
}
