package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Модель актива
type Asset struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Type        string
	Price       float64 // Текущая цена
	Quantity    float64 // Количество в портфеле
	PortfolioID *uint   // ID портфеля, к которому принадлежит актив
}

// Модель портфеля финансовых активов
type Portfolio struct {
	ID     uint    `gorm:"primaryKey"`
	Name   string  `gorm:"unique"`                 // Имя портфеля
	Assets []Asset `gorm:"foreignKey:PortfolioID"` // Связанные активы
}

var db *gorm.DB

// Функция для заполнения базы данных начальными данными
func seedDatabase() {
	// Проверяем, есть ли уже портфели в базе данных
	var portfolioCount int64
	db.Model(&Portfolio{}).Count(&portfolioCount)
	if portfolioCount > 0 {
		fmt.Println("Database already seeded. Skipping...")
		return
	}

	// Создаем начальный портфель
	defaultPortfolio := Portfolio{Name: "Default Portfolio"}
	db.Create(&defaultPortfolio)

	// Добавляем активы в начальный портфель
	assets := []Asset{
		{Name: "Bitcoin", Type: "Cryptocurrency", Price: 1, Quantity: 1, PortfolioID: &defaultPortfolio.ID},
		{Name: "Ethereum", Type: "Cryptocurrency", Price: 1, Quantity: 1, PortfolioID: &defaultPortfolio.ID},
	}

	for _, asset := range assets {
		db.Create(&asset)
	}

	updatePricesNow()

	fmt.Println("Database seeded with default portfolio and assets.")
}

func main() {
	// Инициализация базы данных
	var err error
	db, err = gorm.Open(sqlite.Open("assets.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Asset{})
	db.AutoMigrate(&Portfolio{})

	// Заполнение базы данных начальными данными
	seedDatabase()

	// Запуск периодического обновления цен
	go updatePrices()

	// Middleware для обработки CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Все домены
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			// Обработка предварительных запросов (preflight)
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Регистрация HTTP-обработчиков с middleware
	http.Handle("/add", corsMiddleware(http.HandlerFunc(addAssetHandler)))
	http.Handle("/remove", corsMiddleware(http.HandlerFunc(removeAssetHandler)))
	http.Handle("/get", corsMiddleware(http.HandlerFunc(getPortfolioHandler)))
	http.Handle("/addportfolio", corsMiddleware(http.HandlerFunc(addPortfolioHandler)))
	http.Handle("/getname", corsMiddleware(http.HandlerFunc(getPortfoliosHandler)))

	// Запуск HTTP-сервера
	fmt.Println("Server started on :8080")
	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}
}

// Обновление цен каждые 10 минут
func updatePrices() {
	ticker := time.NewTicker(10 * time.Second * 60)
	defer ticker.Stop() // Остановить тикер при завершении функции

	for range ticker.C {
		var assets []Asset
		db.Find(&assets)

		prices, err := updateCryptoPrices()
		if err != nil {
			fmt.Printf("Error with update crypto: %v", err)
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

		stocks, err := updateStockPrices()
		if err != nil {
			fmt.Printf("Error with update stocks: %v", err)
		}
		for key, value := range stocks {
			prices[key] = value
		}

		// Обновляем цены активов
		for i := range assets {
			if newPrice, exists := prices[assets[i].Name]; exists {
				assets[i].Price = newPrice
				db.Save(&assets[i])
			}
		}

		fmt.Println("Prices updated ", time.Now())
	}
}

func updatePricesNow() {
	var assets []Asset
	db.Find(&assets)

	prices, err := updateCryptoPrices()
	if err != nil {
		fmt.Printf("Error with update crypto: %v", err)
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

	stocks, err := updateStockPrices()
	if err != nil {
		fmt.Printf("Error with update stocks: %v", err)
	}
	for key, value := range stocks {
		prices[key] = value
	}

	// Обновляем цены активов
	for i := range assets {
		if newPrice, exists := prices[assets[i].Name]; exists {
			assets[i].Price = newPrice
			db.Save(&assets[i])
		}
	}

	fmt.Println("Prices updated ", time.Now())
}

func addAssetHandler(w http.ResponseWriter, r *http.Request) {
	var asset Asset
	err := json.NewDecoder(r.Body).Decode(&asset)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверка на существование
	var existingAsset Asset
	result := db.Where("name = ? AND type = ? AND portfolio_id = ?", asset.Name, asset.Type, asset.PortfolioID).First(&existingAsset)
	if result.Error == nil {
		// Актив существует, увеличиваем его количество
		existingAsset.Quantity += asset.Quantity // Предполагается, что у вас есть поле Quantity
		if err := db.Save(&existingAsset).Error; err != nil {
			http.Error(w, fmt.Sprintf("Error updating asset quantity: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Asset quantity updated: %s\n", existingAsset.Name)
	} else if result.Error == gorm.ErrRecordNotFound {
		// Актив не существует, создаем новый

		// Создаем актив
		result := db.Create(&asset)

		// Обновляем цены для всех активов
		addedCrypto()
		addedStock()

		// Перезагружаем актив из базы данных, чтобы получить обновленную цену
		if err := db.Where("id = ?", asset.ID).First(&asset).Error; err != nil {
			http.Error(w, fmt.Sprintf("Error reloading asset: %v", err), http.StatusInternalServerError)
			return
		}

		// Проверяем, что цена не равна 0.0
		if asset.Price == 0.0 && (asset.Type == "Cryptocurrency" || asset.Type == "Stocks") {
			db.Delete(&asset)
			http.Error(w, "Invalid asset: Price cannot be zero", http.StatusBadRequest) // Возвращаем статус 400
			return
		}

		if result.Error != nil {
			http.Error(w, fmt.Sprintf("Error adding asset: %v", result.Error), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// Обработчик удаления актива
func removeAssetHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name        string `json:"name"`
		PortfolioID uint   `json:"portfolioID"`
	}

	// Декодируем тело запроса
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ищем актив по имени и ID портфеля
	var asset Asset
	result := db.Where("name = ? AND portfolio_id = ?", request.Name, request.PortfolioID).Delete(&asset)
	if result.RowsAffected == 0 {
		http.Error(w, fmt.Sprintf("Asset not found: %s in portfolio %d", request.Name, request.PortfolioID), http.StatusNotFound)
		return
	}

	// Возвращаем успешный ответ
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Asset removed: %s", request.Name)})
}

func getPortfolioHandler(w http.ResponseWriter, r *http.Request) {
	portfolioName := r.URL.Query().Get("name")
	if portfolioName == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	var portfolio Portfolio
	result := db.Preload("Assets").Where("name = ?", portfolioName).First(&portfolio)
	if result.RowsAffected == 0 {
		http.Error(w, fmt.Sprintf("Portfolio with name '%s' not found", portfolioName), http.StatusNotFound)
		return
	}

	response := make([]map[string]interface{}, len(portfolio.Assets))
	for i, asset := range portfolio.Assets {
		response[i] = map[string]interface{}{
			"Name":     asset.Name,
			"Type":     asset.Type,
			"Price":    asset.Price,
			"Quantity": asset.Quantity,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": response})
}

// Обработчик добавления нового портфеля
func addPortfolioHandler(w http.ResponseWriter, r *http.Request) {
	var request map[string]string
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	name := request["name"]
	if name == "" {
		http.Error(w, "Portfolio name is required", http.StatusBadRequest)
		return
	}

	// Проверка, существует ли уже портфель с таким именем
	var existingPortfolio Portfolio
	result := db.Where("name = ?", name).First(&existingPortfolio)
	if result.RowsAffected > 0 {
		http.Error(w, fmt.Sprintf("Portfolio with name '%s' already exists", name), http.StatusConflict)
		return
	}

	// Создание нового портфеля
	newPortfolio := Portfolio{Name: name}
	db.Create(&newPortfolio)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Portfolio added: %s\n", name)
}

// Обработчик получения всех портфелей
func getPortfoliosHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр запроса "name"
	portfolioName := r.URL.Query().Get("name")

	var portfolios []Portfolio
	if portfolioName != "" {
		// Если имя портфеля указано, ищем портфель по имени
		db.Preload("Assets").Where("name = ?", portfolioName).Find(&portfolios)
	} else {
		// Если имя не указано, возвращаем все портфели
		db.Preload("Assets").Find(&portfolios)
	}

	// Если портфель не найден, возвращаем ошибку 404
	if len(portfolios) == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Портфель не найден"})
		return
	}

	// Рассчитываем общую стоимость каждого портфеля
	response := make([]map[string]interface{}, len(portfolios))
	for i, portfolio := range portfolios {
		totalBalance := 0.0
		for _, asset := range portfolio.Assets {
			totalBalance += asset.Price * float64(asset.Quantity)
		}

		response[i] = map[string]interface{}{
			"Name":         portfolio.Name,
			"id":           portfolio.ID,
			"totalBalance": totalBalance,
			"assets":       portfolio.Assets,
		}
	}

	// Возвращаем результат
	w.Header().Set("Content-Type", "application/json")
	if portfolioName != "" {
		// Если запрашивался конкретный портфель, возвращаем только его
		json.NewEncoder(w).Encode(response[0])
	} else {
		// Если запрашивались все портфели, возвращаем их список
		json.NewEncoder(w).Encode(map[string]interface{}{"items": response})
	}
}
