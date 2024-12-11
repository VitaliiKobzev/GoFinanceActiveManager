package observer

import (
	"encoding/json"
	"net/http"
)

// Курс валюты
type ForexRate struct {
	Valute map[string]struct {
		Value float64 `json:"Value"`
	} `json:"Valute"`
}

type Value interface {
	GetRubRate(currency string) (float64, error)
}

type ForexService struct{}

func (fs *ForexService) GetRubRate(currency string) (float64, error) {
	url := "https://www.cbr-xml-daily.ru/daily_json.js"
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result ForexRate
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	rate := result.Valute[currency].Value
	return rate, nil
}
