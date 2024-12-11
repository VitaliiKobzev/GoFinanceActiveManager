package observer

import "fmt"

type Customer struct {
	ID string
}

func (c *Customer) Update(itemName string) {
	fmt.Printf("Update to portfolio %s for item %s\n", c.ID, itemName)
}

func (c *Customer) GetID() string {
	return c.ID
}
