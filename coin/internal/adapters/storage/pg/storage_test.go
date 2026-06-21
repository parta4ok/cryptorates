//go:build TEST_L1

package pg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"cryptorates/coin/internal/adapters/storage/pg"
	"cryptorates/coin/internal/entities"
)

var (
	connstr = os.Getenv("COIN_STORAGE")
)

func makeDB(t *testing.T) *pg.Storage {
	t.Helper()

	storage, err := pg.NewStorage(connstr)
	require.NoError(t, err)

	return storage
}

func flushStorage(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := pgxpool.New(ctx, connstr)
	require.NoError(t, err)

	query := `DELETE FROM crypto.coins`
	_, err = db.Exec(ctx, query)
	require.NoError(t, err)
}

func TestStorage_GetAllTitles(t *testing.T) {
	defer flushStorage(t)

	btc, err := entities.NewCoin("BTC", 1.1)
	require.NoError(t, err)

	eth, err := entities.NewCoin("ETH", 0.1)
	require.NoError(t, err)

	storage := makeDB(t)

	ctx := context.Background()
	err = storage.Store(ctx, []*entities.Coin{btc, eth})
	require.NoError(t, err)

	titles, err := storage.GetAllTitles(ctx)
	require.NoError(t, err)

	require.Equal(t, []string{"BTC", "ETH"}, titles)
}

func TestStorage_GetCoins_Succes(t *testing.T) {
	defer flushStorage(t)

	ethLast, err := entities.NewCoin("ETH", 2, entities.WithConcreteActualAt(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	storage := makeDB(t)

	ctx := context.Background()
	err = storage.Store(ctx, []*entities.Coin{ethLast})
	require.NoError(t, err)

	ethCurrent, err := entities.NewCoin("ETH", 1, entities.WithConcreteActualAt(time.Now()))
	require.NoError(t, err)
	err = storage.Store(ctx, []*entities.Coin{ethCurrent})
	require.NoError(t, err)

	coins, err := storage.GetCoins(ctx, []string{"ETH"})
	require.NoError(t, err)

	require.Equal(t, ethLast.Cost(), coins[0].Cost())
}

func TestStorage_GetCoins_NotFound(t *testing.T) {
	defer flushStorage(t)

	storage := makeDB(t)

	ctx := context.Background()

	coins, err := storage.GetCoins(ctx, []string{"ETH"})
	require.ErrorIs(t, err, entities.ErrNotFound)
	require.Nil(t, coins)
}

func TestStorage_GetAggregatedCoins_Succes(t *testing.T) {
	defer flushStorage(t)

	eth1, err := entities.NewCoin("ETH", 2)
	require.NoError(t, err)

	storage := makeDB(t)
	ctx := context.Background()

	err = storage.Store(ctx, []*entities.Coin{eth1})
	require.NoError(t, err)

	eth2, err := entities.NewCoin("ETH", 1)
	require.NoError(t, err)
	err = storage.Store(ctx, []*entities.Coin{eth2})
	require.NoError(t, err)

	coins, err := storage.GetAggregatedCoins(ctx, []string{"ETH"}, "MIN")
	require.NoError(t, err)
	require.Equal(t, eth2.Cost(), coins[0].Cost())

	coins, err = storage.GetAggregatedCoins(ctx, []string{"ETH"}, "MAX")
	require.NoError(t, err)
	require.Equal(t, eth1.Cost(), coins[0].Cost())

	coins, err = storage.GetAggregatedCoins(ctx, []string{"ETH"}, "AVG")
	require.NoError(t, err)
	require.Greater(t, eth1.Cost(), coins[0].Cost())
	require.Less(t, eth2.Cost(), coins[0].Cost())
}
