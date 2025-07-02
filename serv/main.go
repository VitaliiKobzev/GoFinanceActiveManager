package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Модель актива
type Asset struct {
	ID              uint `gorm:"primaryKey"`
	Name            string
	Type            string
	Price           float64 // Текущая цена
	InitialPrice    float64
	Quantity        float64        // Количество в портфеле
	PortfolioID     *uint          // ID портфеля, к которому принадлежит актив
	PriceHistory    []PriceHistory `gorm:"foreignKey:AssetID"` // История цен
	AcquisitionYear *int           // Год приобретения
	ReleaseYear     *int
}

type PriceHistory struct {
	ID        uint `gorm:"primaryKey"`
	AssetID   uint
	Price     float64
	Timestamp time.Time
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
	// Есть ли уже портфели в базе данных
	var portfolioCount int64
	db.Model(&Portfolio{}).Count(&portfolioCount)
	if portfolioCount > 0 {
		fmt.Println("Database already seeded. Skipping...")
		return
	}

	// Начальный портфель
	defaultPortfolio := Portfolio{Name: "Default Portfolio"}
	db.Create(&defaultPortfolio)

	// Активы в начальный портфель
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
	// Параметры подключения к MySQL
	dsn := "dist_admin:XA34619RA3n77L7aZrtK@tcp(127.0.0.1:3306)/assets?charset=utf8mb4&parseTime=True&loc=Local"

	// Инициализация базы данных
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect to MySQL: %v", err))
	}

	// Автомиграция схемы
	err = db.AutoMigrate(&Portfolio{}, &Asset{}, &PriceHistory{})
	if err != nil {
		panic(fmt.Sprintf("failed to auto-migrate database schema: %v", err))
	}
	// Заполнение базы данных начальными данными
	seedDatabase()

	// Инициализация Telegram бота
	tgConfig := TelegramConfig{
		BotToken: botApi,
		ChatID:   chatID,
		Enabled:  true, // Включить/выключить уведомления
	}

	botClient := NewBotClient(tgConfig, db, 0)

	// Запуск периодического обновления цен
	go updatePrices()

	go startHourlyPriceCheck(botClient)

	go botClient.StartDailyNotifications()

	// Middleware для обработки CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // Все домены
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

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

	http.Handle("/editportfolio", corsMiddleware(http.HandlerFunc(editPortfolioHandler)))
	http.Handle("/deleteportfolio", corsMiddleware(http.HandlerFunc(deletePortfolioHandler)))

	http.Handle("/updateinitialprices", corsMiddleware(http.HandlerFunc(updateInitialPricesHandler)))

	http.Handle("/export", corsMiddleware(http.HandlerFunc(exportToExcelHandler)))

	http.Handle("/pricehistory", corsMiddleware(http.HandlerFunc(PriceHistoryHandler)))
	http.Handle("/portfoliohistory", corsMiddleware(http.HandlerFunc(PortfolioHistoryHandler)))
	http.Handle("/addhistory", corsMiddleware(http.HandlerFunc(AddPriceHistoryHandler)))

	http.Handle("/calculate-risk", corsMiddleware(http.HandlerFunc(riskHandler)))

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
			if newPrice, exists := prices[assets[i].Name]; exists {
				// Обновляем текущую цену
				assets[i].Price = newPrice
				db.Save(&assets[i])

				// Добавляем запись в историю цен
				historyEntry := PriceHistory{
					AssetID:   assets[i].ID,
					Price:     newPrice,
					Timestamp: time.Now(),
				}
				db.Create(&historyEntry)
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

		// После успешного создания актива:
		if asset.Price > 0 {
			historyEntry := PriceHistory{
				AssetID:   asset.ID,
				Price:     asset.Price,
				Timestamp: time.Now(),
			}
			db.Create(&historyEntry)
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
			"Name":            asset.Name,
			"Type":            asset.Type,
			"Price":           asset.Price,
			"InitialPrice":    asset.InitialPrice,
			"Quantity":        asset.Quantity,
			"AcquisitionYear": asset.AcquisitionYear,
			"ReleaseYear":     asset.ReleaseYear,
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

			if asset.Price == 0 && asset.Type != "Stocks" && asset.Type != "Cryptocurrency" {
				totalBalance += asset.InitialPrice
			}
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

// Обработчик изменения названия портфеля
func editPortfolioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.OldName == "" || request.NewName == "" {
		http.Error(w, "Both oldName and newName are required", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли портфель с новым именем
	var existingPortfolio Portfolio
	result := db.Where("name = ?", request.NewName).First(&existingPortfolio)
	if result.RowsAffected > 0 {
		http.Error(w, fmt.Sprintf("Portfolio with name '%s' already exists", request.NewName), http.StatusConflict)
		return
	}

	// Находим и обновляем портфель
	var portfolio Portfolio
	result = db.Where("name = ?", request.OldName).First(&portfolio)
	if result.RowsAffected == 0 {
		http.Error(w, fmt.Sprintf("Portfolio with name '%s' not found", request.OldName), http.StatusNotFound)
		return
	}

	portfolio.Name = request.NewName
	db.Save(&portfolio)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Portfolio renamed from '%s' to '%s'", request.OldName, request.NewName),
	})
}

// Обработчик удаления портфеля
func deletePortfolioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Name string `json:"name"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Name == "" {
		http.Error(w, "Portfolio name is required", http.StatusBadRequest)
		return
	}

	// Начинаем транзакцию
	tx := db.Begin()

	// Сначала удаляем все активы портфеля
	if err := tx.Where("portfolio_id IN (SELECT id FROM portfolios WHERE name = ?)", request.Name).Delete(&Asset{}).Error; err != nil {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("Error deleting portfolio assets: %v", err), http.StatusInternalServerError)
		return
	}

	// Затем удаляем сам портфель
	result := tx.Where("name = ?", request.Name).Delete(&Portfolio{})
	if result.RowsAffected == 0 {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("Portfolio with name '%s' not found", request.Name), http.StatusNotFound)
		return
	}

	// Фиксируем транзакцию
	tx.Commit()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Portfolio '%s' and all its assets deleted successfully", request.Name),
	})
}

// updateInitialPricesHandler обрабатывает запрос на обновление начальных цен
func updateInitialPricesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	portfolioName := r.URL.Query().Get("name")
	if portfolioName == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	// Получаем портфель с активами
	var portfolio Portfolio
	result := db.Preload("Assets").Where("name = ?", portfolioName).First(&portfolio)
	if result.Error != nil {
		http.Error(w, fmt.Sprintf("Portfolio not found: %v", result.Error), http.StatusNotFound)
		return
	}

	updatedAssets := updateAssetsInitialPrices(portfolio.Assets)

	if len(updatedAssets) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "No applicable assets to update (only Stocks and Cryptocurrency)"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Initial prices updated successfully",
		"updated": updatedAssets,
	})
}

// updateAssetsInitialPrices обновляет начальные цены для активов
func updateAssetsInitialPrices(assets []Asset) []string {
	updatedAssets := make([]string, 0)

	for i := range assets {
		asset := &assets[i]
		// Обновляем только для акций и криптовалют
		if asset.Type == "Cryptocurrency" || asset.Type == "Stocks" {
			// Устанавливаем начальную цену равной текущей
			asset.InitialPrice = asset.Price * asset.Quantity
			if err := db.Save(asset).Error; err != nil {
				fmt.Printf("Error saving asset: %v", err)
				continue
			}
			updatedAssets = append(updatedAssets, asset.Name)
		}
	}

	return updatedAssets
}

// Получить историю цен для конкретного актива
func PriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из URL
	query := r.URL.Query()
	portfolioName := query.Get("portfolio")
	assetName := query.Get("asset")

	// Логируем полученные параметры для отладки
	log.Printf("Запрос истории цен: портфель=%s, актив=%s", portfolioName, assetName)

	var portfolio Portfolio
	// Ищем портфель и конкретный актив
	if err := db.Where("name = ?", portfolioName).
		Preload("Assets", "name = ?", assetName).
		Preload("Assets.PriceHistory").
		First(&portfolio).Error; err != nil {

		log.Printf("Ошибка поиска: %v", err)
		http.Error(w, "Портфель или актив не найден", http.StatusNotFound)
		return
	}

	if len(portfolio.Assets) == 0 {
		http.Error(w, "Актив не найден", http.StatusNotFound)
		return
	}

	asset := portfolio.Assets[0]

	// Получаем историю цен (последние 30 записей)
	var history []PriceHistory
	if err := db.Where("asset_id = ?", asset.ID).
		Order("timestamp desc").
		Limit(30).
		Find(&history).Error; err != nil {

		http.Error(w, "Ошибка при получении истории", http.StatusInternalServerError)
		return
	}

	// Переворачиваем порядок для графика (от старых к новым)
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// Получить историю стоимости портфеля
func PortfolioHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	portfolioName := query.Get("portfolio")
	daysStr := query.Get("days")

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		days = 0
	}

	var portfolio Portfolio
	if err := db.Where("name = ?", portfolioName).Preload("Assets").Preload("Assets.PriceHistory").First(&portfolio).Error; err != nil {
		http.Error(w, "Портфель не найден", http.StatusNotFound)
		return
	}

	// Собираем все уникальные даты из истории цен всех активов
	dateMap := make(map[time.Time]bool)
	for _, asset := range portfolio.Assets {
		for _, ph := range asset.PriceHistory {
			// Округляем до дня для группировки
			date := time.Date(ph.Timestamp.Year(), ph.Timestamp.Month(), ph.Timestamp.Day(), 0, 0, 0, 0, ph.Timestamp.Location())
			dateMap[date] = true
		}
	}

	// Преобразуем в массив и сортируем
	var dates []time.Time
	for date := range dateMap {
		if days > 0 && time.Since(date) > time.Duration(days)*24*time.Hour {
			continue
		}
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	// Для каждой даты вычисляем общую стоимость портфеля
	type PortfolioValue struct {
		Date       time.Time `json:"Date"`
		TotalValue float64   `json:"TotalValue"`
	}
	var result []PortfolioValue

	for _, date := range dates {
		total := 0.0
		for _, asset := range portfolio.Assets {
			// Ищем последнюю цену актива ДО или НА дату
			var price float64
			found := false

			// Сортируем PriceHistory по убыванию
			sort.Slice(asset.PriceHistory, func(i, j int) bool {
				return asset.PriceHistory[i].Timestamp.After(asset.PriceHistory[j].Timestamp)
			})

			for _, ph := range asset.PriceHistory {
				phDate := time.Date(ph.Timestamp.Year(), ph.Timestamp.Month(), ph.Timestamp.Day(), 0, 0, 0, 0, ph.Timestamp.Location())
				if phDate.Before(date) || phDate.Equal(date) {
					price = ph.Price
					found = true
					break
				}
			}

			if !found {
				price = asset.InitialPrice
			}

			total += price * asset.Quantity
		}

		result = append(result, PortfolioValue{
			Date:       date,
			TotalValue: total,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func AddPriceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	// Разрешаем CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Expected content type application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Читаем тело запроса
	var request struct {
		PortfolioName string    `json:"portfolioName"`
		AssetName     string    `json:"assetName"`
		Price         float64   `json:"price"`
		Timestamp     time.Time `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация данных
	if request.PortfolioName == "" || request.AssetName == "" {
		http.Error(w, "Portfolio name and asset name are required", http.StatusBadRequest)
		return
	}

	if request.Price <= 0 {
		http.Error(w, "Price must be positive", http.StatusBadRequest)
		return
	}

	// Если timestamp не указан, используем текущее время
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	// Находим актив
	var asset Asset
	if err := db.Joins("JOIN portfolios ON portfolios.id = assets.portfolio_id").
		Where("portfolios.name = ? AND assets.name = ?", request.PortfolioName, request.AssetName).
		First(&asset).Error; err != nil {
		return
	}

	// Создаем запись в истории цен
	historyEntry := PriceHistory{
		AssetID:   asset.ID,
		Price:     request.Price,
		Timestamp: request.Timestamp,
	}

	if err := db.Create(&historyEntry).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to create price history: %v", err), http.StatusInternalServerError)
		return
	}

	// Обновляем текущую цену актива
	asset.Price = request.Price
	if err := db.Save(&asset).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to update asset price: %v", err), http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Price history entry added",
		"data":    historyEntry,
	})
}
