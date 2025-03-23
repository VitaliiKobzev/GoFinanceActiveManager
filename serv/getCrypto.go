package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
)

type CMCResponse struct {
	Data map[string]struct {
		Name  string `json:"name"`
		Quote struct {
			USD struct {
				Price float64 `json:"price"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}

func updateCryptoPrices() (map[string]float64, error) {
	// Уникальные криптовалюты из базы данных или новая
	var cryptoAssets []string
	db.Model(&Asset{}).Where("type = ?", "Cryptocurrency").Distinct().Pluck("name", &cryptoAssets)
	for i, asset := range cryptoAssets {
		cryptoAssets[i] = strings.ToLower(asset)
	}
	if len(cryptoAssets) == 0 {
		fmt.Println("Crypto GG")
		return nil, nil // Если нет криптовалют
	}

	url := fmt.Sprintf("https://pro-api.coinmarketcap.com/v2/cryptocurrency/quotes/latest?slug=%s", strings.Join(cryptoAssets, ","))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)
	req.Header.Add("Accepts", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return nil, fmt.Errorf("error fetching data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var result CMCResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error unmarshalling response: %v", err)
		return nil, fmt.Errorf("error processing response: %w", err)
	}

	prices := make(map[string]float64)
	for _, data := range result.Data {
		price := data.Quote.USD.Price
		prices[data.Name] = float64(math.Round(price*100) / 100)
	}

	return prices, nil
}

func addedCrypto() {
	var assets []Asset
	db.Find(&assets)

	prices, err := updateCryptoPrices()
	if err != nil {
		fmt.Printf("Error getting CryptoPrices: %v", err)
		return
	}
	fs := &ForexService{}
	rubRate, err := fs.GetRubRate("USD")
	if err != nil {
		fmt.Printf("Error getting RUB rate: %v", err)
		return
	}
	for key, value := range prices {
		if len(key) > 0 {
			newKey := strings.ToUpper(string(key[0])) + key[1:]
			prices[newKey] = value * rubRate
		}
	}

	// Обновляем цены активов
	for i := range assets {
		if newPrice, exists := prices[assets[i].Name]; exists {
			assets[i].Price = newPrice
			db.Save(&assets[i])
		}
	}

	fmt.Println("Added crypto")
}
