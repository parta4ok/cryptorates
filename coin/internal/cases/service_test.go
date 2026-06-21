package cases_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"cryptorates/coin/internal/cases"
	"cryptorates/coin/internal/cases/testdata"
	"cryptorates/coin/internal/entities"
)

var (
	errTest = errors.New("test error")
)

func Test_NewService_Failure(t *testing.T) {
	t.Parallel()

	var nilStorage cases.Storage
	var nilProvider cases.Provider

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	provider := testdata.NewMockProvider(ctrl)
	service, err := cases.NewService(nilProvider, nilStorage)
	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Nil(t, service)

	service, err = cases.NewService(provider, nilStorage)
	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Nil(t, service)
}

func TestService_GetActualRates(t *testing.T) {
	t.Parallel()

	type stages struct {
		stageStorageGetAllTitles    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, err error)
		stageStorageGetAllTitlesErr error

		stageProviderGetActualRates    func(ctx context.Context, t *testing.T, p *testdata.MockProvider, titles []string, coins []*entities.Coin, err error)
		stageProviderGetActualRatesErr error

		stageStorageStore    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, coins []*entities.Coin, err error)
		stageStorageStoreErr error

		stageStorageGetCoins    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, coins []*entities.Coin, err error)
		stageStorageGetCoinsErr error
	}
	tests := []struct {
		name    string
		stages  stages
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageStorageGetAllTitlesErr: errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stages: stages{
				stageStorageGetAllTitles:       stageStorageGetAllTitles,
				stageProviderGetActualRates:    stageProviderGetActualRates,
				stageProviderGetActualRatesErr: errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageProviderGetActualRates: stageProviderGetActualRates,
				stageStorageStore:           stageStorageStore,
				stageStorageStoreErr:        errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "4",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageProviderGetActualRates: stageProviderGetActualRates,
				stageStorageStore:           stageStorageStore,
				stageStorageGetCoins:        stageStorageGetCoins,
				stageStorageGetCoinsErr:     errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "5",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageProviderGetActualRates: stageProviderGetActualRates,
				stageStorageStore:           stageStorageStore,
				stageStorageGetCoins:        stageStorageGetCoins,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			storage := testdata.NewMockStorage(ctrl)
			provider := testdata.NewMockProvider(ctrl)

			service, err := cases.NewService(provider, storage)
			require.NoError(it, err)
			require.NotNil(it, service)

			ctx := context.TODO()
			titleBTC := "BTC"
			titleETH := "ETH"

			requiredTitles := []string{titleBTC, titleETH}
			savedTitles := []string{titleBTC}
			unsavedTitles := []string{titleETH}

			coinBTC, err := entities.NewCoin(titleBTC, 1.1)
			require.NoError(it, err)

			coinETH, err := entities.NewCoin(titleETH, 0.1)
			require.NoError(it, err)

			unsavedCoins := []*entities.Coin{coinETH}
			expectedCoins := []*entities.Coin{coinBTC, coinETH}

			if tc.stages.stageStorageGetAllTitles != nil {
				tc.stages.stageStorageGetAllTitles(ctx, it, storage, savedTitles, tc.stages.stageStorageGetAllTitlesErr)
			}

			if tc.stages.stageProviderGetActualRates != nil {
				tc.stages.stageProviderGetActualRates(ctx, it, provider, unsavedTitles, unsavedCoins, tc.stages.stageProviderGetActualRatesErr)
			}

			if tc.stages.stageStorageStore != nil {
				tc.stages.stageStorageStore(ctx, it, storage, unsavedCoins, tc.stages.stageStorageStoreErr)
			}

			if tc.stages.stageStorageGetCoins != nil {
				tc.stages.stageStorageGetCoins(ctx, it, storage, requiredTitles, expectedCoins, tc.stages.stageStorageGetCoinsErr)
			}

			resultCoins, err := service.GetActualRates(ctx, requiredTitles)
			if tc.wantErr {
				require.Nil(it, resultCoins)
				require.ErrorIs(it, err, tc.resErr)
				return
			}
			require.NoError(it, err)
			require.Equal(it, expectedCoins, resultCoins)
		})
	}
}

func TestService_GetAggregateRates(t *testing.T) {
	t.Parallel()

	type stages struct {
		stageStorageGetAllTitles    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, err error)
		stageStorageGetAllTitlesErr error

		stageProviderGetActualRates    func(ctx context.Context, t *testing.T, p *testdata.MockProvider, titles []string, coins []*entities.Coin, err error)
		stageProviderGetActualRatesErr error

		stageStorageStore    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, coins []*entities.Coin, err error)
		stageStorageStoreErr error

		stageStorageGetAggregatedCoins    func(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, aggType string, coins []*entities.Coin, err error)
		stageStorageGetAggregatedCoinsErr error
	}
	tests := []struct {
		name    string
		stages  stages
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageStorageGetAllTitlesErr: errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stages: stages{
				stageStorageGetAllTitles:       stageStorageGetAllTitles,
				stageProviderGetActualRates:    stageProviderGetActualRates,
				stageProviderGetActualRatesErr: errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stages: stages{
				stageStorageGetAllTitles:    stageStorageGetAllTitles,
				stageProviderGetActualRates: stageProviderGetActualRates,
				stageStorageStore:           stageStorageStore,
				stageStorageStoreErr:        errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "4",
			stages: stages{
				stageStorageGetAllTitles:          stageStorageGetAllTitles,
				stageProviderGetActualRates:       stageProviderGetActualRates,
				stageStorageStore:                 stageStorageStore,
				stageStorageGetAggregatedCoins:    stageStorageGetAggregatedCoins,
				stageStorageGetAggregatedCoinsErr: errTest,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "5",
			stages: stages{
				stageStorageGetAllTitles:       stageStorageGetAllTitles,
				stageProviderGetActualRates:    stageProviderGetActualRates,
				stageStorageStore:              stageStorageStore,
				stageStorageGetAggregatedCoins: stageStorageGetAggregatedCoins,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			storage := testdata.NewMockStorage(ctrl)
			provider := testdata.NewMockProvider(ctrl)

			service, err := cases.NewService(provider, storage)
			require.NoError(it, err)
			require.NotNil(it, service)

			ctx := context.TODO()
			aggType := "max"
			titleBTC := "BTC"
			titleETH := "ETH"

			requiredTitles := []string{titleBTC, titleETH}
			savedTitles := []string{titleBTC}
			unsavedTitles := []string{titleETH}

			coinBTC, err := entities.NewCoin(titleBTC, 1.1)
			require.NoError(it, err)

			coinETH, err := entities.NewCoin(titleETH, 0.1)
			require.NoError(it, err)

			unsavedCoins := []*entities.Coin{coinETH}
			expectedCoins := []*entities.Coin{coinBTC, coinETH}

			if tc.stages.stageStorageGetAllTitles != nil {
				tc.stages.stageStorageGetAllTitles(ctx, it, storage, savedTitles, tc.stages.stageStorageGetAllTitlesErr)
			}

			if tc.stages.stageProviderGetActualRates != nil {
				tc.stages.stageProviderGetActualRates(ctx, it, provider, unsavedTitles, unsavedCoins, tc.stages.stageProviderGetActualRatesErr)
			}

			if tc.stages.stageStorageStore != nil {
				tc.stages.stageStorageStore(ctx, it, storage, unsavedCoins, tc.stages.stageStorageStoreErr)
			}

			if tc.stages.stageStorageGetAggregatedCoins != nil {
				tc.stages.stageStorageGetAggregatedCoins(ctx, it, storage, requiredTitles, aggType, expectedCoins, tc.stages.stageStorageGetAggregatedCoinsErr)
			}

			resultCoins, err := service.GetAggregateRates(ctx, requiredTitles, aggType)
			if tc.wantErr {
				require.Nil(it, resultCoins)
				require.ErrorIs(it, err, tc.resErr)
				return
			}
			require.NoError(it, err)
			require.Equal(it, expectedCoins, resultCoins)
		})
	}
}

func stageStorageGetAllTitles(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, err error) {
	t.Helper()

	s.EXPECT().GetAllTitles(ctx).Return(titles, err)
}

func stageProviderGetActualRates(ctx context.Context, t *testing.T, p *testdata.MockProvider, titles []string, coins []*entities.Coin, err error) {
	t.Helper()

	p.EXPECT().GetActualRates(ctx, titles).Return(coins, err)
}

func stageStorageStore(ctx context.Context, t *testing.T, s *testdata.MockStorage, coins []*entities.Coin, err error) {
	t.Helper()

	s.EXPECT().Store(ctx, coins).Return(err)
}

func stageStorageGetCoins(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, coins []*entities.Coin, err error) {
	t.Helper()

	s.EXPECT().GetCoins(ctx, titles).Return(coins, err)
}

func stageStorageGetAggregatedCoins(ctx context.Context, t *testing.T, s *testdata.MockStorage, titles []string, aggType string, coins []*entities.Coin, err error) {
	t.Helper()

	s.EXPECT().GetAggregatedCoins(ctx, titles, aggType).Return(coins, err)
}

type StageFn func(t *testing.T, provider *testdata.MockProvider, storage *testdata.MockStorage)

func TestService_ActualizeRates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stages  []StageFn
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			stages: []StageFn{
				stageGetAllTitlesWithErr,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "2",
			stages: []StageFn{
				stageGetAllTitlesWithSucces,
				stageGetActualRatesWithErr,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "3",
			stages: []StageFn{
				stageGetAllTitlesWithSucces,
				stageGetActualRatesWithSuccess,
				stgeStoreWithErr,
			},
			wantErr: true,
			resErr:  errTest,
		},
		{
			name: "4",
			stages: []StageFn{
				stageGetAllTitlesWithSucces,
				stageGetActualRatesWithSuccess,
				stgeStoreWithSuccess,
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			storage := testdata.NewMockStorage(ctrl)
			provider := testdata.NewMockProvider(ctrl)

			service, err := cases.NewService(provider, storage)
			require.NoError(t, err)
			require.NotNil(t, service)

			for _, stage := range tc.stages {
				stage(t, provider, storage)
			}

			ctx := context.TODO()

			err = service.ActualizeRates(ctx)
			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				return
			}

			require.NoError(it, err)
		})
	}
}

func stageGetAllTitlesWithErr(t *testing.T, _ *testdata.MockProvider, storage *testdata.MockStorage) {
	t.Helper()

	storage.EXPECT().GetAllTitles(gomock.Any()).Return(nil, errTest)
}

func stageGetAllTitlesWithSucces(t *testing.T, _ *testdata.MockProvider, storage *testdata.MockStorage) {
	t.Helper()

	storage.EXPECT().GetAllTitles(gomock.Any()).Return([]string{"BTC", "ETH"}, nil)
}

func stageGetActualRatesWithErr(t *testing.T, provider *testdata.MockProvider, _ *testdata.MockStorage) {
	t.Helper()

	provider.EXPECT().GetActualRates(gomock.Any(), []string{"BTC", "ETH"}).Return(nil, errTest)
}

func stageGetActualRatesWithSuccess(t *testing.T, provider *testdata.MockProvider, _ *testdata.MockStorage) {
	t.Helper()

	coinBTC, err := entities.NewCoin("BTC", 1.1)
	require.NoError(t, err)

	coinETH, err := entities.NewCoin("ETH", 0.1)
	require.NoError(t, err)

	provider.EXPECT().GetActualRates(gomock.Any(), []string{"BTC", "ETH"}).Return([]*entities.Coin{coinBTC, coinETH}, nil)
}

func stgeStoreWithErr(t *testing.T, _ *testdata.MockProvider, storage *testdata.MockStorage) {
	t.Helper()

	storage.EXPECT().Store(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coins []*entities.Coin) error {
			if len(coins) == 2 &&
				(coins[0].Title() == "ETH" || coins[0].Title() == "BTC") &&
				(coins[1].Title() == "ETH" || coins[1].Title() == "BTC") {
				return errTest
			}
			return errors.New("other error")
		})
}

func stgeStoreWithSuccess(t *testing.T, _ *testdata.MockProvider, storage *testdata.MockStorage) {
	t.Helper()

	storage.EXPECT().Store(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, coins []*entities.Coin) error {
			if len(coins) == 2 &&
				(coins[0].Title() == "ETH" || coins[0].Title() == "BTC") &&
				(coins[1].Title() == "ETH" || coins[1].Title() == "BTC") {
				return nil
			}
			return errors.New("other error")
		})
}
