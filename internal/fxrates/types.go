package fxrates

import (
	"github.com/sig-0/fxrates/server"
	"github.com/sig-0/fxrates/storage/types"
)

// Convenience
type (
	ExchangeRate       = types.ExchangeRate
	PageExchangeRate   = types.Page[types.ExchangeRate]
	Currency           = types.Currency
	Source             = types.Source
	RateType           = types.RateType
	SourcesResponse    = server.SourcesResponse
	CurrenciesResponse = server.CurrenciesResponse
)
