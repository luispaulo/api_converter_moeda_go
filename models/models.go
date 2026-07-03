package models

import "time"

// ExchangeRate representa o modelo de taxas de câmbio salvas no banco de dados
type ExchangeRate struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Rate      float64   `json:"rate"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConversionLog representa o histórico de conversão gravado no banco de dados
type ConversionLog struct {
	ID              int       `json:"id"`
	AmountBRL       float64   `json:"amount_brl"`
	TargetCurrency  string    `json:"target_currency"`
	Rate            float64   `json:"rate"`
	ConvertedAmount float64   `json:"converted_amount"`
	ConvertedAt     time.Time `json:"converted_at"`
}

// AwesomeAPICurrencyInfo mapeia o JSON de retorno individual de cada par da AwesomeAPI
type AwesomeAPICurrencyInfo struct {
	Code       string `json:"code"`
	CodeIn     string `json:"codein"`
	Name       string `json:"name"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

// ConversionResult representa o resultado final de uma conversão para resposta da API
type ConversionResult struct {
	Currency        string  `json:"currency"`
	Name            string  `json:"name"`
	Rate            float64 `json:"rate"`
	ConvertedAmount float64 `json:"converted_amount"`
}
