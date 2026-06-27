//go:build TEST_L2

package l2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cryptorates/coin/pkg/dto"
)

type TestClient struct {
	*http.Client
}

func NewClient(t *testing.T, timeout time.Duration) *TestClient {
	t.Helper()

	client := &http.Client{
		Timeout: timeout,
	}

	return &TestClient{
		Client: client,
	}
}

func sendRequestToCoin(t *testing.T, aggregatedType string, titles ...string) *http.Response {
	t.Helper()

	client := NewClient(t, 3*time.Second)

	var urlRaw *url.URL
	var err error

	if aggregatedType == "" {
		urlRaw, err = url.Parse("http://localhost:8080/crypto/v1/rates")
		require.NoError(t, err)
	} else {
		urlRaw, err = url.Parse(fmt.Sprintf("%s/%s", "http://localhost:8080/crypto/v1/rates",
			aggregatedType))
		require.NoError(t, err)
	}

	requestedTitles := dto.TitlesDTO{Titles: append([]string{}, titles...)}
	data, err := json.Marshal(requestedTitles)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, urlRaw.String(),
		bytes.NewReader(data))
	require.NoError(t, errReq)

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

type CoinDTO struct {
	Title    string    `json:"title"`
	Cost     float64   `json:"cost"`
	ActualAt time.Time `actual_at:""`
}

type CoinsDTO struct {
	Coins []CoinDTO `json:"coins"`
}

func Test_Coin_Success(t *testing.T) {
	t.Parallel()

	coins := []string{"btc", "eth"}

	resp := sendRequestToCoin(t, "", coins...)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	var coinsDTO CoinsDTO
	err := json.NewDecoder(resp.Body).Decode(&coinsDTO)
	require.NoError(t, err)

	respCoins := make([]string, 0, len(coins))
	for _, coin := range coinsDTO.Coins {
		respCoins = append(respCoins, coin.Title)
		require.Greater(t, coin.Cost, 0.0)
		require.Greater(t, time.Now(), coin.ActualAt)
	}

	require.ElementsMatch(t, coins, respCoins)
}
