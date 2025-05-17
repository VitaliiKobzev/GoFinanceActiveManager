package main

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"gorm.io/gorm"
)

// настройки для Telegram бота
type TelegramConfig struct {
	BotToken string
	ChatID   string
	Enabled  bool
}

// клиент для работы с Telegram ботом
type BotClient struct {
	config              TelegramConfig
	lastPortfolioValues map[uint]float64
	changeLimit         float64
	db                  *gorm.DB
}

// экземпляр BotClient
func NewBotClient(config TelegramConfig, db *gorm.DB, changeLimit float64) *BotClient {
	return &BotClient{
		config:              config,
		lastPortfolioValues: make(map[uint]float64),
		changeLimit:         changeLimit,
		db:                  db,
	}
}

// отправка
func (b *BotClient) SendNotification(message string) error {
	if !b.config.Enabled {
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.config.BotToken)
	data := url.Values{
		"chat_id":    {b.config.ChatID},
		"text":       {message},
		"parse_mode": {"HTML"},
	}

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("error sending Telegram message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Telegram message, status code: %d", resp.StatusCode)
	}

	return nil
}

func startHourlyPriceCheck(b *BotClient) {
	b.CheckPortfolioChanges()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		b.CheckPortfolioChanges()
	}
}

// CheckPortfolioChanges проверяет значительные изменения в стоимости портфелей
func (b *BotClient) CheckPortfolioChanges() {
	var portfolios []Portfolio
	if err := b.db.Preload("Assets").Find(&portfolios).Error; err != nil {
		fmt.Printf("Error fetching portfolios: %v\n", err)
		return
	}

	// Флаг первого запуска
	isFirstRun := b.lastPortfolioValues == nil

	// Инициализация при первом запуске
	if isFirstRun {
		b.lastPortfolioValues = make(map[uint]float64)
	}

	// Проверяем изменения для каждого портфеля
	for _, portfolio := range portfolios {
		currentValue := calculatePortfolioValue(portfolio.Assets)

		if isFirstRun {
			// При первом запуске просто сохраняем текущее значение
			b.lastPortfolioValues[portfolio.ID] = currentValue

			// Отправляем начальное уведомление (если changeLimit = 0)
			if b.changeLimit == 0 {
				message := fmt.Sprintf(
					"🆕 <b>Инициализация портфеля %s</b>\n"+
						"Начальная стоимость: <b>%.2f ₽</b>",
					portfolio.Name,
					currentValue,
				)
				if err := b.SendNotification(message); err != nil {
					fmt.Printf("Error sending notification: %v\n", err)
				}
			}
			continue
		}

		lastValue, exists := b.lastPortfolioValues[portfolio.ID]
		if !exists {
			// Новый портфель - сохраняем и отправляем уведомление
			b.lastPortfolioValues[portfolio.ID] = currentValue
			message := fmt.Sprintf(
				"🆕 <b>Добавлен новый портфель %s</b>\n"+
					"Начальная стоимость: <b>%.2f ₽</b>",
				portfolio.Name,
				currentValue,
			)
			if err := b.SendNotification(message); err != nil {
				fmt.Printf("Error sending notification: %v\n", err)
			}
			continue
		}

		if lastValue == 0 {
			// Избегаем деления на ноль
			continue
		}

		changePercent := (currentValue - lastValue) / lastValue * 100
		absChange := math.Abs(changePercent)

		if b.changeLimit == 0 || absChange >= b.changeLimit {
			// Формируем сообщение
			trend := "📈"
			if changePercent < 0 {
				trend = "📉"
			}

			message := fmt.Sprintf(
				"%s <b>Изменение портфеля %s</b> %s\n"+
					"Изменение: <b>%.2f%%</b>\n"+
					"Предыдущая стоимость: <b>%.2f ₽</b>\n"+
					"Текущая стоимость: <b>%.2f ₽</b>",
				trend,
				portfolio.Name,
				trend,
				changePercent,
				lastValue,
				currentValue,
			)

			// Отправляем уведомление
			if err := b.SendNotification(message); err != nil {
				fmt.Printf("Error sending notification: %v\n", err)
			}
		}

		// Обновляем последнее значение
		b.lastPortfolioValues[portfolio.ID] = currentValue
	}
}

// Вспомогательная функция для расчета стоимости портфеля
func calculatePortfolioValue(assets []Asset) float64 {
	total := 0.0
	for _, asset := range assets {
		total += asset.Price * asset.Quantity
	}
	return total
}

// SendDailyPortfolioReport отправляет ежедневный отчет
func (b *BotClient) SendDailyPortfolioReport() {
	var portfolios []Portfolio
	b.db.Preload("Assets").Find(&portfolios)

	if len(portfolios) == 0 {
		return
	}

	message := "<b>Ежедневный отчет по портфелям</b>\n\n"
	totalAllPortfolios := 0.0

	for _, portfolio := range portfolios {
		portfolioTotal := 0.0
		initialTotal := 0.0
		for _, asset := range portfolio.Assets {
			portfolioTotal += asset.Price * asset.Quantity
			initialTotal += asset.InitialPrice
		}
		totalAllPortfolios += portfolioTotal

		changePercent := 0.0
		if initialTotal > 0 {
			changePercent = (portfolioTotal - initialTotal) / initialTotal * 100
		}

		message += fmt.Sprintf(
			"<b>%s</b>\n"+
				"Текущая стоимость: <b>%.2f ₽</b>\n"+
				"Изменение с начала: <b>%.2f%%</b>\n\n",
			portfolio.Name,
			portfolioTotal,
			changePercent,
		)
	}

	message += fmt.Sprintf("\n<b>Общая стоимость всех портфелей: %.2f ₽</b>", totalAllPortfolios)

	if err := b.SendNotification(message); err != nil {
		fmt.Printf("Error sending daily Telegram notification: %v\n", err)
	}
}

// StartDailyNotifications запускает ежедневные уведомления
func (b *BotClient) StartDailyNotifications() {

	b.SendDailyPortfolioReport()

	// Запускаем в 17:00 каждый день
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(next.Sub(now))

		b.SendDailyPortfolioReport()
	}
}
