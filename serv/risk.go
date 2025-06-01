package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sync"
)

type RiskResult struct {
	Level string  `json:"level"`
	Value float64 `json:"value"`
}

type VolatilityData struct {
	Volatility float64 `json:"volatility"`
}

func CalculatePortfolioRisk(assets []Asset) (*RiskResult, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	// 1. Получаем данные о волатильности
	volatilityData, err := FetchAssetVolatility(assets)
	if err != nil {
		return nil, err
	}

	// 2. Рассчитываем общую стоимость портфеля
	totalValue := 0.0
	for _, asset := range assets {
		totalValue += asset.Price
	}

	if totalValue == 0 {
		return nil, nil
	}

	// 3. Расчет дисперсии портфеля
	portfolioVariance := 0.0

	for i := 0; i < len(assets); i++ {
		asset1 := assets[i]
		weight1 := asset1.Price / totalValue
		volatility1 := volatilityData[asset1.Type].Volatility

		// Вклад в общий риск
		portfolioVariance += math.Pow(weight1, 2) * math.Pow(volatility1, 2)

		// Учитываем корреляцию с другими активами
		for j := i + 1; j < len(assets); j++ {
			asset2 := assets[j]
			weight2 := asset2.Price / totalValue
			volatility2 := volatilityData[asset2.Type].Volatility
			correlation := GetCorrelation(asset1.Type, asset2.Type)

			// Ковариация
			portfolioVariance += 2 * weight1 * weight2 * correlation * volatility1 * volatility2
		}
	}

	// Общий риск портфеля
	portfolioRisk := math.Sqrt(portfolioVariance)

	// 4. Определяем уровень риска
	riskLevel := ""
	switch {
	case portfolioRisk < 0.1:
		riskLevel = "Консервативный"
	case portfolioRisk < 0.2:
		riskLevel = "Умеренно-консервативный"
	case portfolioRisk < 0.3:
		riskLevel = "Умеренный"
	case portfolioRisk < 0.5:
		riskLevel = "Агрессивный"
	default:
		riskLevel = "Очень агрессивный"
	}

	return &RiskResult{
		Level: riskLevel,
		Value: portfolioRisk,
	}, nil
}

func FetchAssetVolatility(assets []Asset) (map[string]VolatilityData, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	volatilityData := make(map[string]VolatilityData)
	errChan := make(chan error, len(assets))
	done := make(chan bool)

	for _, asset := range assets {
		wg.Add(1)
		go func(a Asset) {
			defer wg.Done()

			var volatility float64
			var err error

			switch a.Type {
			case "Stocks", "Bonds":
				volatility, err = FetchMOEXVolatility(a.Name)
				if err != nil {
					errChan <- err
					return
				}
			case "Cryptocurrency":
				volatility, err = FetchCryptoVolatility(a.Name)
				if err != nil {
					errChan <- err
					return
				}
			default:
				volatility = 0.15
			}

			mu.Lock()
			volatilityData[a.Type] = VolatilityData{Volatility: volatility}
			mu.Unlock()
		}(asset)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return volatilityData, nil
	case err := <-errChan:
		return nil, err
	}
}

func FetchMOEXVolatility(ticker string) (float64, error) {
	resp, err := http.Get(fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/shares/securities/%s/volatility.json", ticker))
	if err != nil {
		return 0.2, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.2, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0.2, err
	}

	volatility := 0.2
	if val, ok := result["volatility"].(float64); ok {
		volatility = val
	}

	return volatility, nil
}

func FetchCryptoVolatility(ticker string) (float64, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://pro-api.coinmarketcap.com/v2/cryptocurrency/ohlcv/historical?symbol=%s", ticker), nil)
	if err != nil {
		return 0.5, err
	}

	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return 0.5, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0.5, err
	}

	var data struct {
		Data struct {
			Prices []struct {
				Close float64 `json:"close"`
			} `json:"prices"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return 0.5, err
	}

	prices := data.Data.Prices
	if len(prices) < 2 {
		return 0.5, nil
	}

	// Рассчитываем логарифмическую доходность и волатильность
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = math.Log(prices[i].Close / prices[i-1].Close)
	}

	sumSq := 0.0
	for _, r := range returns {
		sumSq += r * r
	}

	volatility := math.Sqrt(sumSq / float64(len(returns)))
	return volatility, nil
}

func GetCorrelation(assetType1, assetType2 string) float64 {
	correlationMatrix := map[string]map[string]float64{
		"Stocks": {
			"Stocks":         1,
			"Bonds":          -0.2,
			"Cryptocurrency": 0.1,
		},
		"Bonds": {
			"Stocks":         -0.2,
			"Bonds":          1,
			"Cryptocurrency": -0.1,
		},
		"Cryptocurrency": {
			"Stocks":         0.1,
			"Bonds":          -0.1,
			"Cryptocurrency": 1,
		},
	}

	if matrix, ok := correlationMatrix[assetType1]; ok {
		if corr, ok := matrix[assetType2]; ok {
			return corr
		}
	}

	return 0
}

func riskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var assets []Asset
	if err := json.NewDecoder(r.Body).Decode(&assets); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := CalculatePortfolioRisk(assets)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calculating risk: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
