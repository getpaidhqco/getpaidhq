package cart

import "fmt"

type CreateCartOptions struct {
	Currency string
	Items    []Item
}

func New(options CreateCartOptions) Cart {
	return Cart{
		Currency: options.Currency,
		Total:    0,
		SubTotal: 0,
		Discount: 0,
		Shipping: 0,
		Tax:      0,
		Items:    options.Items,
	}
}

func (c *Cart) AddItem(item Item) (*Cart, error) {
	c.Items = append(c.Items, item)
	return c.Calculate()
}

func (c *Cart) AdjustQuantity(id string, quantity int64) (*Cart, error) {
	for i, item := range c.Items {
		if item.ID == id {
			c.Items[i].Quantity = quantity
			return c.Calculate()
		}
	}
	return nil, fmt.Errorf("item not found")
}

func (c *Cart) RemoveItem(id string) (*Cart, error) {
	for i, item := range c.Items {
		if item.ID == id {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			break
		}
	}
	return c.Calculate()
}

func (c *Cart) Calculate() (*Cart, error) {
	var total int64 = 0
	var subTotal int64 = 0

	for _, item := range c.Items {
		subTotal += item.Price.UnitPrice * item.Quantity
		total = subTotal
	}
	c.Total = total
	c.SubTotal = subTotal
	return c, nil
}
