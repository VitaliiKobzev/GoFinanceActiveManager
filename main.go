package main

import (
	"log"
	"net/http"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("assets.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&Asset{})
}

func main() {
	initDB()
	http.HandleFunc("/add", addAsset)
	http.HandleFunc("/delete", delAsset)
	http.HandleFunc("/assets", getAssets)
	http.HandleFunc("/updateCrypto", updateCryptoPrices)
	http.HandleFunc("/updateStock", updateStockPrices)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Println("Server started at :2233")
	http.ListenAndServe(":2233", nil)
}
