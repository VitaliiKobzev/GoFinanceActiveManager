# GoFinanceActiveMonitoring
 
Локальное приложение целью которого является мониторинг по активам пользователя.

## Функционал
- **Множество портфелей.**
- **Уведомления в мессенджер.**
- **Автоматическое обновление цен.**
- **Подключение к API Мосбиржи.**
- **Подключение к API CoinMarketCap.**
- **Возможность изменять портфели.**
- **Получение истории стоимости портфеля.**
- **Получение истории цен на актив.**
- **Делегирование серверной и клиентской части.**
- **Переход на MySQL.**
- **Множество страниц.**
- **Получение таблицы активов.**
- **Добавление цен на активы вручную.**
- **Получение таблицы активов с указанием года выпуска и годом приобретения актива.**
- **Удаление части активов при неподдержке**

## Запуск
Для запуска нужно создать в serv файл apiKeys.go, который должен выглядеть ориентировочно так:
```golang
package main

var apiKey = "abc...xyz"
var botApi = "abc...xyz"
var chatID = "123456789"
```
В serv/main.go изменить логин и пароль пользователя MySQL:
```golang
dsn := "log:pass@tcp(127.0.0.1:3306)/assets?charset=utf8mb4&parseTime=True&loc=Local"
```
Запуск серверной части:
```console
go run serv/main.go serv/getCrypto.go serv/apiKeys.go serv/getCurrency.go serv/stock.go serv/bot.go serv/excel.go serv/risk.go
```
Запуск клиентской части:
```console
go run client/main.go
```
### Скриншоты приложения
![Главная страница](https://i.ibb.co/JjbCRFDC/34537-799x350.png)
![Страница портфеля](https://i.ibb.co/5gvrVTTr/34538-800x319.png)
![Страница изменения портфеля](https://i.ibb.co/Qv60q0R3/34539-434x156.png)