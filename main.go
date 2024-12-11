package main

import (
	obs "active/observer"
	"fmt"
	"log"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db          *gorm.DB
	portfolios  []*obs.Customer
	Items       []obs.Item
	curPortofio int = 0
)

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("items.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&obs.Item{})
	//db.AutoMigrate(&Asset{})
}

func main() {
	initDB()
	//shirtItem := obs.NewItem("Bitcoin", 90000, 0.005, "USD")
	portfolios = append(portfolios, &obs.Customer{ID: "Test"})
	fmt.Println(portfolios[curPortofio].GetID())
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
