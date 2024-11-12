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

func updateStockPrices(w http.ResponseWriter, r *http.Request) {
	var stockAssets []string
	db.Model(&Asset{}).Where("category = ?", "Stocks").Distinct().Pluck("name", &stockAssets)
	for i, asset := range stockAssets {
		stockAssets[i] = strings.ToUpper(asset)
	}
	//fmt.Println(stockAssets)
	if len(stockAssets) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	for _, asset := range stockAssets {
		url := fmt.Sprintf("https://iss.moex.com/iss/engines/stock/markets/shares/boards/TQBR/securities/%s.xml", asset)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error fetching data for %s: %v\n", asset, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error response for %s: %s\n", asset, resp.Status)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body for %s: %v\n", asset, err)
			continue
		}

		var stockResponse StockResponse
		if err := xml.Unmarshal(body, &stockResponse); err != nil {
			fmt.Printf("Error decoding response for %s: %v\n", asset, err)
			continue
		}

		for _, data := range stockResponse.Data {
			if data.ID == "marketdata" {
				for _, row := range data.Rows.Row {
					if row.SecID == asset {
						//fmt.Printf("Stock: %s\n", row.SecID)
						//fmt.Printf("Last Price: %.2f\n", row.Last)
						var existingAsset Asset
						if db.Where("name = ? AND category = ?", row.SecID, "Stocks").First(&existingAsset).Error == nil {
							existingAsset.Cost = math.Round(row.Last*existingAsset.Quantity*100) / 100
							db.Save(&existingAsset)
							//log.Printf("Updated stock asset: %+v\n", existingAsset)
						}
					}
				}
			}
		}
	}

	fmt.Fprintf(w, "Stock prices updated successfully")
}
