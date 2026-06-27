package cryptocompare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"cryptorates/coin/internal/adapters/provider/baseclient"
	"cryptorates/coin/internal/entities"
	"cryptorates/tools/tracing/tracer"
)

const (
	baseURL        = "https://min-api.cryptocompare.com"
	priceMultiPath = "/data/pricemulti"

	queryFsyms = "fsyms"
	queryTsyms = "tsyms"
)

var (
	_ baseclient.RatesSource = (*Client)(nil)
)

type Client struct{}

func NewClient() (*Client, error) {
	c := &Client{}

	return c, nil
}

func (c *Client) GetRequestData(
	ctx context.Context, titles []string, costIn string) (*http.Request, error) {
	slog.Info("GetRequestData")
	ctx, span, cancel := tracer.Start(ctx, "GetRequestData for cryprocompare") //nolint:ineffassign,lll,staticcheck //ok
	defer cancel()

	url, err := url.Parse(fmt.Sprintf("%s%s", baseURL, priceMultiPath))
	if err != nil {
		err := errors.Wrap(entities.ErrInternal, "url parse")
		span.SetError(err)
		return nil, err
	}

	query := url.Query()
	query.Set(queryFsyms, strings.Join(titles, ","))
	query.Set(queryTsyms, costIn)
	url.RawQuery = query.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil) //nolint:wrapcheck //ok
}

func (c *Client) ParseResponseData(
	ctx context.Context, body io.Reader, costIn string) ([]*entities.Coin, error) {
	slog.Info("ParseResponseData")
	ctx, span, cancel := tracer.Start(ctx, "ParseResponseData for cryprocompare") //nolint:ineffassign,lll,staticcheck //ok
	defer cancel()

	var respMap map[string]map[string]float64

	if err := json.NewDecoder(body).Decode(&respMap); err != nil {
		err := errors.Wrap(entities.ErrInternal, "json decode")
		span.SetError(err)
		return nil, err
	}

	coins := make([]*entities.Coin, 0)
	if len(respMap) == 0 {
		err := errors.Wrap(entities.ErrNotFound, "result not contained data")
		span.SetError(err)
		return nil, err
	}
	for title, costMap := range respMap {
		coin, err := entities.NewCoin(title, costMap[costIn])
		if err != nil {
			err := errors.Wrap(err, "new coin from response")
			span.SetError(err)
			return nil, err
		}
		coins = append(coins, coin)
	}

	return coins, nil
}
