package entities

import (
	"time"

	"github.com/pkg/errors"
)

type Coin struct {
	title    string
	cost     float64
	actualAt time.Time
}

type CoinOption func(*Coin)

func WithConcreteActualAt(actualAt time.Time) CoinOption {
	return func(c *Coin) {
		c.actualAt = actualAt
	}
}

func (c *Coin) setOptions(opts ...CoinOption) {
	for _, opt := range opts {
		opt(c)
	}
}

func NewCoin(title string, cost float64, opts ...CoinOption) (*Coin, error) {
	coin := &Coin{
		title:    title,
		cost:     cost,
		actualAt: time.Now(),
	}

	coin.setOptions(opts...)

	if coin.title == "" {
		return nil, errors.Wrap(ErrInvalidParam, "title not set")
	}

	if coin.cost < 0.0 {
		return nil, errors.Wrap(ErrInvalidParam, "cost must be greater then 0.0")
	}

	if coin.actualAt.IsZero() {
		return nil, errors.Wrap(ErrInvalidParam, "time not set")
	}

	return coin, nil
}

func (c *Coin) Title() string {
	return c.title
}

func (c *Coin) Cost() float64 {
	return c.cost
}

func (c *Coin) ActualAt() time.Time {
	return c.actualAt
}
