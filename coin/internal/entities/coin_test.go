package entities_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cryptorates/coin/internal/entities"
)

func TestNewCoin(t *testing.T) {
	t.Parallel()

	var nilTime time.Time
	type args struct {
		title string
		cost  float64
		opts  []entities.CoinOption
	}
	tests := []struct {
		name    string
		args    args
		want    *entities.Coin
		wantErr bool
		resErr  error
	}{
		{
			name: "invalit title",
			args: args{
				title: "",
				cost:  1.1,
				opts:  nil,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "invalit cost",
			args: args{
				title: "BTC",
				cost:  -1.1,
				opts:  nil,
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "invalit actualAt",
			args: args{
				title: "BTC",
				cost:  1.1,
				opts:  []entities.CoinOption{entities.WithConcreteActualAt(nilTime)},
			},
			wantErr: true,
			resErr:  entities.ErrInvalidParam,
		},
		{
			name: "success",
			args: args{
				title: "BTC",
				cost:  1.1,
				opts:  []entities.CoinOption{entities.WithConcreteActualAt(getConcreteTime(t))},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			coin, err := entities.NewCoin(tc.args.title, tc.args.cost, tc.args.opts...)
			if tc.wantErr {
				require.Nil(it, coin)
				require.ErrorIs(it, err, tc.resErr)
				return
			}
			require.NotNil(it, coin)
			require.NoError(it, err)

			require.Equal(it, tc.args.title, coin.Title())
			require.Equal(it, tc.args.cost, coin.Cost())
			require.Equal(it, getConcreteTime(it), coin.ActualAt())
		})
	}
}
func getConcreteTime(t *testing.T) time.Time {
	t.Helper()

	concreteTime, err := time.Parse(time.RFC3339, "2026-02-13T20:50:00+05:00")
	require.NoError(t, err)

	return concreteTime
}
