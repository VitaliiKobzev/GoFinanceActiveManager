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

func updateCryptoPrices(w http.ResponseWriter, r *http.Request) {
	// Уникальные криптовалюты из базы данных
	var cryptoAssets []string
	db.Model(&Asset{}).Where("category = ?", "Cryptocurrency").Distinct().Pluck("name", &cryptoAssets)
	for i, asset := range cryptoAssets {
		cryptoAssets[i] = strings.ToLower(asset)
	}
	//fmt.Println(cryptoAssets)
	if len(cryptoAssets) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	//https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s&key=%s
	url := fmt.Sprintf("https://pro-api.coinmarketcap.com/v2/cryptocurrency/quotes/latest?slug=%s", strings.Join(cryptoAssets, ","))
	req, _ := http.NewRequest("GET", url, nil)
	// req.Header.Add("Content-Type", "application/json")
	// req.Header.Add("Accept", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", apiKey)
	req.Header.Add("Accepts", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		http.Error(w, "Error fetching data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		http.Error(w, "Error reading response body", http.StatusInternalServerError)
		return
	}
	//fmt.Println(string(body))

	var result CMCResponse
	json.Unmarshal(body, &result)
	//fmt.Println(result)
	for _, data := range result.Data {
		price := data.Quote.USD.Price
		//fmt.Printf("Цена - %f\n", price)
		var existingAsset Asset
		if db.Where("name = ? AND category = ?", data.Name, "Cryptocurrency").First(&existingAsset).Error == nil {
			toRub, err := getRubRate("USD")
			if err != nil {
				log.Print("Error with USD\n")
				return
			}
			existingAsset.Cost = math.Round(price*existingAsset.Quantity*toRub*100) / 100
			db.Save(&existingAsset)
			//log.Printf("Updated crypto asset: %+v\n", existingAsset)
		}
	}

	w.WriteHeader(http.StatusOK)
}
