package observer

import (
	"fmt"
	"math"
	"time"
)

type Item struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ObserverList []Observer `gorm:"-" json:"-"`
	Category     string     `json:"category"`
	Name         string     `json:"name"`
	Cost         float64    `json:"cost"`
	Income       float64    `json:"income"`
	Expense      float64    `json:"expense"`
	Quantity     float64    `json:"quantity"`
	TotalPrice   float64    `json:"total_price"`
	Valuta       string     `json:"value"`
	Portfolio    uint       `json:"portfolio"`
	CreatedAt    time.Time  `json:"created_at"`
}

func NewItem(name string, category string, cost float64, income float64, expense float64, quantity float64, totalPrice float64, valuta string, portfolio uint) *Item {
	return &Item{
		Name:       name,
		Category:   category,
		Cost:       cost,
		Income:     income,
		Expense:    expense,
		Quantity:   quantity,
		TotalPrice: totalPrice,
		Valuta:     valuta,
		Portfolio:  portfolio,
		CreatedAt:  time.Now(),
	}
}

func (i *Item) UpdateAvailability(costNew float64) {
	fmt.Printf("Item %s is updated as %.2f\n", i.Name, costNew)
	i.Cost = math.Round(costNew*100) / 100
	//i.TotalPrice = float64(i.Cost * i.Quantity)
	//i.notifyAll()
}
func (i *Item) Register(o Observer) {
	i.ObserverList = append(i.ObserverList, o)
}

func (i *Item) Update(u Item) {
	i.Cost += u.Cost
	i.Income += u.Income
	i.Expense += u.Expense
	i.Quantity += u.Quantity
}

func (i *Item) deregister(o Observer) {
	i.ObserverList = removeFromslice(i.ObserverList, o)
}

func (i *Item) notifyAll() {
	for _, observer := range i.ObserverList {
		observer.Update(i.Name)
	}
}

func removeFromslice(observerList []Observer, observerToRemove Observer) []Observer {
	observerListLength := len(observerList)
	for i, observer := range observerList {
		if observerToRemove.GetID() == observer.GetID() {
			observerList[observerListLength-1], observerList[i] = observerList[i], observerList[observerListLength-1]
			return observerList[:observerListLength-1]
		}
	}
	return observerList
}
