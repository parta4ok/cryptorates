package coingecco

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
	baseURL         = "https://api.coingecko.com/api/v3"
	simplePricePath = "/simple/price"

	authorizationQuery = "x_cg_demo_api_key"

	symbolsQuery      = "symbols"
	vsCurrenciesQuery = "vs_currencies"
)

var (
	_ baseclient.RatesSource = (*Client)(nil)
)

type Client struct {
	token string
}

func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "token not set")
	}
	c := &Client{
		token: token,
	}

	return c, nil
}

func (c *Client) GetRequestData(
	ctx context.Context, titles []string, costIn string) (*http.Request, error) {
	slog.Info("GetRequestData coingecco")
	ctx, span, cancel := tracer.Start(ctx, "GetRequestData for coingecco") //nolint:ineffassign,lll,staticcheck //ok
	defer cancel()

	url, err := url.Parse(fmt.Sprintf("%s%s", baseURL, simplePricePath))
	if err != nil {
		err := errors.Wrap(entities.ErrInternal, "url parse")
		span.SetError(err)
		return nil, err
	}

	query := url.Query()
	query.Set(symbolsQuery, strings.Join(titles, ","))
	query.Set(vsCurrenciesQuery, costIn)
	query.Set(authorizationQuery, c.token)
	url.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		err := errors.Wrap(err, "new request with context")
		span.SetError(err)
		return nil, err
	}

	slog.Info(url.String())

	return req, nil
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
