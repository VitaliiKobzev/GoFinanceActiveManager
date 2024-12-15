package observer

import "fmt"

type Customer struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
}

func (c *Customer) Update(itemName string) {
	fmt.Printf("Update to portfolio %s for item %s\n", c.Name, itemName)
}

func (c *Customer) GetName() string {
	return c.Name
}
