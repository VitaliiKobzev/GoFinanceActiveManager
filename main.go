package main

import (
	obs "active/server"
	"log"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
	//portfolios  []*obs.Customer
	Items       []obs.Item
	curPortofio uint = 0
)

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("items.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&obs.Item{})
	db.AutoMigrate(&obs.Customer{})
	//db.AutoMigrate(&Asset{})
}

func addFirstItemIfEmpty() {
	var count int64
	db.Model(&obs.Item{}).Count(&count)

	if count == 0 {
		firstItem := obs.Customer{Name: "Example"}
		if err := db.Create(&firstItem).Error; err != nil {
			log.Fatalf("failed to add first item: %v", err)
		}
		log.Println("Первое портфолио добавлено:", firstItem.GetName())
	}
}

func main() {
	initDB()
	addFirstItemIfEmpty()
	//shirtItem := obs.NewItem("Bitcoin", 90000, 0.005, "USD")
	// shirtItem.Register(portfolio)
	// shirtItem.UpdateAvailability()
	http.HandleFunc("/add", addItem)
	http.HandleFunc("/delete", delItem)
	http.HandleFunc("/assets", getItems)
	//http.HandleFunc("/updateCrypto", updateCryptoPrices)
	//http.HandleFunc("/updateStock", updateStockPrices)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Println("Server started at :2233")
	http.ListenAndServe(":2233", nil)
}
