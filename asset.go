package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Asset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Cost      float64   `json:"cost"`
	Income    float64   `json:"income"`
	Expense   float64   `json:"expense"`
	Quantity  float64   `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

type AssetInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Cost     string `json:"cost"`
	Income   string `json:"income"`
	Expense  string `json:"expense"`
	Quantity string `json:"quantity"`
}

func addAsset(w http.ResponseWriter, r *http.Request) {
	var input AssetInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		log.Printf("Error decoding JSON: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cost, err := strconv.ParseFloat(input.Cost, 64)
	if err != nil {
		log.Printf("Error converting cost: %v\n", err)
		http.Error(w, "Invalid cost value", http.StatusBadRequest)
		return
	}

	income, err := strconv.ParseFloat(input.Income, 64)
	if err != nil {
		log.Printf("Error converting income: %v\n", err)
		http.Error(w, "Invalid income value", http.StatusBadRequest)
		return
	}

	expense, err := strconv.ParseFloat(input.Expense, 64)
	if err != nil {
		log.Printf("Error converting expense: %v\n", err)
		http.Error(w, "Invalid expense value", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.ParseFloat(input.Quantity, 64)
	if err != nil {
		log.Printf("Error converting quantity: %v\n", err)
		http.Error(w, "Invalid quantity value", http.StatusBadRequest)
		return
	}

	var existingAsset Asset
	if result := db.Where("name = ? AND category = ?", input.Name, input.Category).First(&existingAsset); result.Error == nil {
		existingAsset.Cost += cost
		existingAsset.Income += income
		existingAsset.Expense += expense
		existingAsset.Quantity += quantity
		db.Save(&existingAsset)
		log.Printf("Updated asset: %+v\n", existingAsset)
		json.NewEncoder(w).Encode(existingAsset)
	} else {
		asset := Asset{
			Name:      input.Name,
			Category:  input.Category,
			Cost:      cost,
			Income:    income,
			Expense:   expense,
			Quantity:  quantity,
			CreatedAt: time.Now(),
		}
		db.Create(&asset)
		//log.Printf("Added asset: %+v\n", asset)
		json.NewEncoder(w).Encode(asset)
	}
}

func delAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Asset name is required", http.StatusBadRequest)
		return
	}

	var asset Asset
	if result := db.Where("name = ?", name).Delete(&asset); result.Error != nil {
		log.Printf("Error deleting asset: %v\n", result.Error)
		http.Error(w, "Error deleting asset", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted asset: %s\n", name)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Asset %s deleted successfully", name)
}

func getAssets(w http.ResponseWriter, r *http.Request) {
	var assets []Asset
	db.Find(&assets)
	var response []map[string]interface{}
	for _, asset := range assets {
		response = append(response, map[string]interface{}{
			"name":     asset.Name,
			"category": asset.Category,
			"value":    asset.Cost,
			"quantity": asset.Quantity,
		})
	}
	//log.Printf("Assets data: %+v\n", response) // Добавим вывод отладочной информации
	json.NewEncoder(w).Encode(response)
}
