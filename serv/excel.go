package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func exportToExcelHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем все портфели с активами
	var portfolios []Portfolio
	if err := db.Preload("Assets").Find(&portfolios).Error; err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получения портфелей: %v", err), http.StatusInternalServerError)
		return
	}

	// Создаем новый файл Excel
	f := excelize.NewFile()

	// Для каждого портфеля создаем отдельный лист
	for _, portfolio := range portfolios {
		// Создаем название листа на основе имени портфеля
		sheetName := sanitizeSheetName(portfolio.Name)
		if sheetName == "" {
			sheetName = fmt.Sprintf("Портфель %d", portfolio.ID)
		}

		// Создаем новый лист
		index, err := f.NewSheet(sheetName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка создания листа: %v", err), http.StatusInternalServerError)
			return
		}

		// Устанавливаем русские заголовки
		headers := []string{"Название", "Тип", "Цена", "Количество", "Общая стоимость"}
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}

		// Заполняем данные активов
		row := 2
		for _, asset := range portfolio.Assets {
			data := []interface{}{
				asset.Name,
				asset.Type,
				asset.Price,
				asset.Quantity,
				asset.Price * asset.Quantity,
			}

			for col, value := range data {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				f.SetCellValue(sheetName, cell, value)
			}
			row++
		}

		// Добавляем итоговую строку только если есть активы
		if len(portfolio.Assets) > 0 {
			totalRowLabel := "Итого"
			f.SetCellValue(sheetName, "A"+strconv.Itoa(row), totalRowLabel)

			// Устанавливаем формулу для суммирования столбца E (Общая стоимость)
			formula := fmt.Sprintf("SUM(E2:E%d)", row-1)
			f.SetCellFormula(sheetName, "E"+strconv.Itoa(row), formula)
		}

		// Стиль для заголовков и итоговой строки
		boldStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			NumFmt:    4, // Формат числа с двумя десятичными знаками
		})

		// Применяем стиль к заголовкам
		f.SetCellStyle(sheetName, "A1", "E1", boldStyle)

		// Если есть активы, применяем стиль к итоговой строке
		if len(portfolio.Assets) > 0 {
			f.SetCellStyle(sheetName, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row), boldStyle)
		}

		// Форматирование числовых столбцов
		numStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 4}) // Формат с 2 знаками после запятой
		if len(portfolio.Assets) > 0 {
			f.SetCellStyle(sheetName, "C2", "E"+strconv.Itoa(row), numStyle)
		}

		// Автоматическая ширина столбцов
		for _, col := range []string{"A", "B", "C", "D", "E"} {
			f.SetColWidth(sheetName, col, col, 18)
		}

		// Делаем текущий лист активным
		f.SetActiveSheet(index)
	}

	// Удаляем лист по умолчанию (Sheet1)
	f.DeleteSheet("Sheet1")

	// Устанавливаем заголовки для скачивания файла
	w.Header().Set("Content-Disposition", "attachment; filename=portfolios_export.xlsx")
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	// Пишем файл в ответ
	if err := f.Write(w); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка отправки Excel файла: %v", err), http.StatusInternalServerError)
	}
}

// Функция для очистки имени листа от недопустимых символов
func sanitizeSheetName(name string) string {
	if name == "" {
		return ""
	}

	// Excel имеет ограничения на имена листов
	invalidChars := []string{":", "\\", "/", "?", "*", "[", "]"}
	for _, char := range invalidChars {
		name = strings.ReplaceAll(name, char, "")
	}

	// Ограничение длины имени (31 символ в Excel)
	if len(name) > 31 {
		name = name[:31]
	}

	return name
}
