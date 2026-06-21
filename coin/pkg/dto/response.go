package dto

import "time"

type CoinDTO struct {
	Title    string    `json:"title"`
	Cost     float64   `json:"cost"`
	ActualAt time.Time `json:"actual_at"`
}

type CoinsDTO struct {
	Coins []CoinDTO `json:"coins"`
}

type ErrorDTO struct {
	Status  int    `json:"status_code"`
	Message string `json:"message,omitempty"`
}
