package main

import (
	obs "active/server"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
)

// type Asset struct {
// 	ID        uint      `gorm:"primaryKey" json:"id"`
// 	Name      string    `json:"name"`
// 	Category  string    `json:"category"`
// 	Cost      float64   `json:"cost"`
// 	Income    float64   `json:"income"`
// 	Expense   float64   `json:"expense"`
// 	Quantity  float64   `json:"quantity"`
// 	CreatedAt time.Time `json:"created_at"`
// }

type ItemInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Cost     string `json:"cost"`
	Income   string `json:"income"`
	Expense  string `json:"expense"`
	Quantity string `json:"quantity"`
}

func addItem(w http.ResponseWriter, r *http.Request) {
	var input ItemInput
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

	var existingItem obs.Item
	if result := db.Where("name = ? AND category = ?", input.Name, input.Category).First(&existingItem); result.Error == nil {
		existingItem.Update(obs.Item{
			Cost:     cost,
			Income:   income,
			Expense:  expense,
			Quantity: quantity,
		})
		// Сохранение обновленного элемента
		if err := db.Save(&existingItem).Error; err != nil {
			log.Printf("Error saving updated asset: %v\n", err)
			http.Error(w, "Error saving updated asset", http.StatusInternalServerError)
			return
		}

		log.Printf("Updated asset: %+v\n", existingItem)
		json.NewEncoder(w).Encode(existingItem)
	} else {
		newItem := obs.NewItem(input.Name, input.Category, float64(cost), income, expense, quantity, float64(cost), "RUB", curPortofio)

		//log
		var customers []obs.Customer
		if err := db.Find(&customers).Error; err != nil {
			log.Fatalf("failed to query customers: %v", err)
		}
		currentPortfolioLog := customers[curPortofio]
		log.Printf("Current Portfolio ID: %+v\n", currentPortfolioLog)

		db.Create(&newItem)
		//Items = append(Items, *newItem)

		// for i := range Items {
		// 	fmt.Print(Items[i].Name)
		// }
		//log.Printf("Added asset: %+v\n", asset)
		json.NewEncoder(w).Encode(newItem)
	}
}

func delItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Item name is required", http.StatusBadRequest)
		return
	}

	var item obs.Item
	if result := db.Where("name = ?", name).Delete(&item); result.Error != nil {
		log.Printf("Error deleting item: %v\n", result.Error)
		http.Error(w, "Error deleting item", http.StatusInternalServerError)
		return
	}

	log.Printf("Deleted item: %s\n", name)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Asset %s deleted successfully", name)
}

// Получение цен + стоимости портфеля
func getItems(w http.ResponseWriter, r *http.Request) {
	var items []obs.Item
	db.Find(&items)
	prices, err := updateCryptoPrices()
	if err != nil {
		log.Printf("Error with update crypto: %v", err)
	}
	stocks, err := updateStockPrices()
	if err != nil {
		log.Printf("Error with update stocks: %v", err)
	}
	for key, value := range stocks {
		prices[key] = value
	}
	for key, value := range prices {
		fmt.Printf("Key: %s, Value: %.2f\n", key, value)
	}
	if err != nil {
		log.Printf("Error updating crypto prices: %v", err)
		return
	}
	for i := range items {
		item := &items[i]
		price, exists := prices[item.Name]
		if exists {
			fmt.Println("Category", item.Category)
			if item.Category == "Cryptocurrency" {
				fs := &obs.ForexService{}

				rubRate, err := fs.GetRubRate("USD")
				if err != nil {
					log.Printf("Error getting RUB rate: %v", err)
					return
				}
				item.UpdateAvailability(float64(math.Round(price*100)/100) * rubRate)
				fmt.Printf("%.2f\n", float64(math.Round(price*100)/100)*rubRate)
			} else {
				item.UpdateAvailability(float64(math.Round(price*100) / 100))
			}

			// Сохраняем обновлённый объект в базу данных
			if err := db.Save(item).Error; err != nil {
				log.Printf("Error saving updated item %s: %v", item.Name, err)
			}
		} else {
			log.Printf("Price not found for item: %s", item.Name)
		}
	}
	var response []map[string]interface{}
	totalBalance := 0.0
	// for _, item := range Items {
	// 	totalBalance += float64(item.Cost) * float64(item.Quantity)
	// }
	for _, item := range items {
		response = append(response, map[string]interface{}{
			"name":     item.Name,
			"category": item.Category,
			"value":    item.Cost * item.Quantity,
			"quantity": item.Quantity,
		})
		//fmt.Println(item.Cost, item.Quantity, "<---- Собственно")
		totalBalance += item.Cost * item.Quantity
	}

	result := map[string]interface{}{
		"items":        response,
		"totalBalance": totalBalance,
	}

	json.NewEncoder(w).Encode(result)

	//log.Printf("Assets data: %+v\n", response) // отладочная информация
}
