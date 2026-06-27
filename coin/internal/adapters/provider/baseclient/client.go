package baseclient

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"cryptorates/coin/internal/cases"
	"cryptorates/coin/internal/entities"
	"cryptorates/tools/tracing/tracer"
)

const (
	defaultCostIn = "usd"
)

var (
	_ cases.Provider = (*BaseClient)(nil)
)

type BaseClient struct {
	*http.Client
	ratesSource RatesSource
	costIn      string
}

type ClientOption func(*BaseClient)

func WithCustomCostIn(costIn string) ClientOption {
	return func(c *BaseClient) {
		c.costIn = costIn
	}
}

func WithCustomTimeOut(timeout time.Duration) ClientOption {
	return func(c *BaseClient) {
		c.Timeout = timeout
	}
}

func (c *BaseClient) setOpts(opts ...ClientOption) {
	for _, opt := range opts {
		opt(c)
	}
}

func NewBaseClient(ratesSource RatesSource, opts ...ClientOption) (*BaseClient, error) {
	if ratesSource == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "rates source not set")
	}

	baseClient := &BaseClient{
		Client:      &http.Client{},
		ratesSource: ratesSource,
		costIn:      defaultCostIn,
	}

	baseClient.setOpts(opts...)

	if baseClient.costIn == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "costIn not set")
	}

	return baseClient, nil
}

func (bc *BaseClient) GetActualRates(
	ctx context.Context, titles []string) ([]*entities.Coin, error) {
	slog.Info("GetActualRates started")
	ctx, span, cancel := tracer.Start(ctx, "GetActualRates")
	defer cancel()

	req, err := bc.ratesSource.GetRequestData(ctx, titles, bc.costIn)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "new request data err: %v", err)
		span.SetError(err)
		return nil, err
	}

	resp, err := bc.Do(req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "request do err: %v", err)
		span.SetError(err)
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck //ok

	var respErr error

	switch resp.StatusCode {
	case http.StatusOK:
		return bc.ratesSource.ParseResponseData(ctx, resp.Body, bc.costIn) //nolint:wrapcheck //ok
	case http.StatusBadRequest:
		respErr = entities.ErrInvalidParam
	case http.StatusNotFound:
		respErr = entities.ErrNotFound
	default:
		respErr = entities.ErrInternal
	}

	respErr = errors.Wrapf(respErr, "provider status: %s", resp.Status)
	span.SetError(respErr)
	return nil, respErr
}
