package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
)

type StockData struct {
	SecID string  `xml:"SECID,attr"`
	Last  float64 `xml:"LAST,attr"`
	Open  float64 `xml:"OPEN,attr"`
	High  float64 `xml:"HIGH,attr"`
	Low   float64 `xml:"LOW,attr"`
}

// оборачивает row
type Rows struct {
	Row []StockData `xml:"row"`
}

// оборачивает data
type Data struct {
	ID   string `xml:"id,attr"`
	Rows Rows   `xml:"rows"`
}

// Ответ от API Московской биржи
type StockResponse struct {
	Data []Data `xml:"data"`
}

func updateStockPrices() (map[string]float64, error) {
	var stockAssets []string
	db.Model(&Asset{}).Where("type = ?", "Stocks").Distinct().Pluck("name", &stockAssets)
	for i, asset := range stockAssets {
		stockAssets[i] = strings.ToUpper(asset)
	}
	if len(stockAssets) == 0 {
		fmt.Println("Stock GG")
		return nil, nil
	}

	prices := make(map[string]float64)

	for _, item := range stockAssets {
		url := fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/%s.xml", item)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error fetching data for %s: %v\n", item, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error response for %s: %s\n", item, resp.Status)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body for %s: %v\n", item, err)
			continue
		}

		var stockResponse StockResponse
		if err := xml.Unmarshal(body, &stockResponse); err != nil {
			fmt.Printf("Error decoding response for %s: %v\n", item, err)
			continue
		}

		for _, data := range stockResponse.Data {
			if data.ID == "marketdata" {
				for _, row := range data.Rows.Row {
					if row.SecID == item {
						//fmt.Printf("Stock: %s\n", row.SecID)
						//fmt.Printf("Last Price: %.2f\n", row.Last)
						var existingAsset Asset
						if db.Where("name = ? AND type = ?", row.SecID, "Stocks").First(&existingAsset).Error == nil {
							fmt.Println(existingAsset.Quantity, ' ', existingAsset.Price)
							existingAsset.Price = float64(math.Round(row.Last*100) / 100)
							price := float64(math.Round(row.Last*100) / 100)
							prices[row.SecID] = price
							db.Save(&existingAsset)
							//log.Printf("Updated stock asset: %+v\n", existingAsset)
						} else {
							fmt.Println(db.Where("name = ? AND type = ?", row.SecID, "Stocks").First(&existingAsset).Error)
						}
					}
				}
			}
		}
	}
	fmt.Println("Stock prices updated successfully", prices)
	return prices, nil
}

func addedStock() {
	var assets []Asset
	if err := db.Find(&assets).Error; err != nil {
		fmt.Printf("Error fetching assets: %v\n", err)
		return
	}

	prices, err := updateStockPrices()
	if err != nil {
		fmt.Printf("Error getting stock prices: %v\n", err)
		return
	}

	for i := range assets {
		if newPrice, exists := prices[assets[i].Name]; exists {
			fmt.Printf("Updating price for %s from %.2f to %.2f\n", assets[i].Name, assets[i].Price, newPrice)
			assets[i].Price = newPrice

			if err := db.Save(&assets[i]).Error; err != nil {
				fmt.Printf("Error saving asset %s: %v\n", assets[i].Name, err)
				continue
			}
		} else {
			fmt.Printf("No price found for %s\n", assets[i].Name)
		}
	}

	fmt.Println("Added stock")
}
