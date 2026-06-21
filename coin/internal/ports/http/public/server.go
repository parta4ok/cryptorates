package public

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"

	"cryptorates/coin/internal/entities"
	"cryptorates/coin/internal/ports"
	"cryptorates/coin/pkg/dto"
	"cryptorates/tools/metrics"
	"cryptorates/tools/tracing/tracer"
)

const (
	basePath  = "/crypto/v1"
	ratesPath = "/rates"

	aggregationType     = "aggregation_type"
	aggregationUrlParam = "{" + aggregationType + "}"
)

type Server struct {
	router  *http.Server
	service ports.Service
}

func NewServer(service ports.Service, port string, timeout time.Duration) (*Server, error) {
	if service == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "service not set")
	}

	if port == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "port not set")
	}

	if timeout == 0 {
		return nil, errors.Wrap(entities.ErrInvalidParam, "timeout not set")
	}

	return &Server{
		router: &http.Server{
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
			Addr:         fmt.Sprintf(":%s", port),
		},
		service: service,
	}, nil
}

func (s *Server) Start() error {
	s.registerRoutes()

	go func() {
		if err := s.router.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "err", err.Error())
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("server will be stopping")

	if err := s.router.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "error", err)
		return err
	}

	return nil
}

func (s *Server) registerRoutes() {
	router := chi.NewRouter()

	router.Use(metrics.HTTPMetricsMiddleware)
	router.Use(s.timeoutMiddleware)

	router.Post(fmt.Sprintf("%s%s", basePath, ratesPath), s.actualRates)
	router.Post(fmt.Sprintf("%s%s/%s", basePath, ratesPath, aggregationUrlParam), s.aggregatedRates)

	s.router.Handler = router
}

// @Summary      Getting rates of requested cryptocurrencies
// @Description  Returns the rates of the requested cryptocurrencies in JSON format.
// @Tags         crypto
// @Accept       json
// @Produce      json
// @Param        request body dto.TitlesDTO true "list of required titles"
// @Success      200  {object}  dto.CoinsDTO "list of coins"
// @Failure      400  {object}  dto.ErrorDTO "bad request"
// @Failure      404  {object}  dto.ErrorDTO "not found"
// @Failure      500  {object}  dto.ErrorDTO "internal server error"
// @Router       /rates [post]
func (s *Server) actualRates(resp http.ResponseWriter, req *http.Request) {
	slog.Info("requested actual rates")
	ctx, span, cancel := tracer.Start(req.Context(), "actualRatesHandler")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	var titles dto.TitlesDTO
	err := json.NewDecoder(req.Body).Decode(&titles)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "decode body failure: %v", err)
		span.SetError(err)
		s.errProcessing(err, resp)
		return
	}

	coins, err := s.service.GetActualRates(ctx, titles.Titles)
	if err != nil {
		err := errors.Wrap(err, "get actual rates failed")
		span.SetError(err)
		s.errProcessing(err, resp)
		return
	}

	s.processCoins(resp, coins)
}

// @Summary      Getting aggregated rates of requested cryptocurrencies
// @Description  Returns aggregated the rates of the requested cryptocurrencies in JSON format.
// @Tags         crypto
// @Accept       json
// @Produce      json
// @Param        aggregation_type path string true "aggregation type" Enums(min, max, avg)
// @Param        request body dto.TitlesDTO true "list of required titles"
// @Success      200  {object}  dto.CoinsDTO "list of coins"
// @Failure      400  {object}  dto.ErrorDTO "bad request"
// @Failure      404  {object}  dto.ErrorDTO "not found"
// @Failure      500  {object}  dto.ErrorDTO "internal server error"
// @Router       /rates/{aggregation_type} [post]
func (s *Server) aggregatedRates(resp http.ResponseWriter, req *http.Request) {
	slog.Info("requested aggregated rates")
	ctx, span, cancel := tracer.Start(req.Context(), "aggregatedRatesHandler")
	defer cancel()

	resp.Header().Set("Content-Type", "application/json")

	aggTypeRaw := chi.URLParam(req, aggregationType)
	aggType, ok := normalizeAggType(aggTypeRaw)
	if !ok {
		err := errors.Wrapf(entities.ErrInvalidParam, "aggregation type not correct: %s", aggTypeRaw)
		span.SetError(err)
		s.errProcessing(err, resp)
		return
	}

	var titles dto.TitlesDTO
	err := json.NewDecoder(req.Body).Decode(&titles)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "decode body failure: %v", err)
		span.SetError(err)
		s.errProcessing(err, resp)
		return
	}

	coins, err := s.service.GetAggregateRates(ctx, titles.Titles, aggType)
	if err != nil {
		err := errors.Wrap(err, "get actual rates failed")
		span.SetError(err)
		s.errProcessing(err, resp)
		return
	}

	s.processCoins(resp, coins)
}

func (s *Server) processCoins(resp http.ResponseWriter, coins []*entities.Coin) {
	respCoins := dto.CoinsDTO{
		Coins: make([]dto.CoinDTO, 0, len(coins)),
	}

	for _, coin := range coins {
		respCoins.Coins = append(respCoins.Coins, dto.CoinDTO{
			Title:    coin.Title(),
			Cost:     coin.Cost(),
			ActualAt: coin.ActualAt(),
		})
	}

	data, err := json.Marshal(&respCoins)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal coins failure: %v", err)
		s.errProcessing(err, resp)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(data) //nolint:errcheck, gosec //skip
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.router.WriteTimeout)
		defer cancel()

		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})
}

func (s *Server) errProcessing(err error, resp http.ResponseWriter) {
	statusCode := http.StatusInternalServerError

	errDTO := dto.ErrorDTO{
		Status:  statusCode,
		Message: err.Error(),
	}

	switch {
	case errors.Is(err, entities.ErrInternal):
		errDTO.Status = http.StatusInternalServerError
	case errors.Is(err, entities.ErrInvalidParam):
		errDTO.Status = http.StatusBadRequest
	case errors.Is(err, entities.ErrNotFound):
		errDTO.Status = http.StatusNotFound
	default:
		errDTO.Status = http.StatusInternalServerError
	}

	data, err := json.Marshal(&errDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error("marshalling failure", "err", err)
		return
	}

	resp.WriteHeader(errDTO.Status)
	resp.Write(data) //nolint:errcheck,gosec //skip
}

func normalizeAggType(agg string) (string, bool) {
	switch strings.ToLower(agg) {
	case "min", "max", "avg":
		return strings.ToUpper(agg), true
	default:
		return "", false
	}
}
