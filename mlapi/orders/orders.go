package orders

import (
	"encoding/json"
	"fmt"
)

type OrderItem struct {
	ItemID        string  `json:"item_id"`
	CategoryID    string  `json:"category_id"`
	Quantity      int     `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	ListingTypeID string  `json:"listing_type_id"`
	SaleFee       float64 `json:"sale_fee"`
}

type Order struct {
	OrderID      int64       `json:"order_id"`
	Status       string      `json:"status"`
	DateCreated  string      `json:"date_created"`
	ShippingCost float64     `json:"shipping_cost"`
	Items        []OrderItem `json:"items"`
}

func ExtractOrders(data []byte) ([]Order, error) {
	var raw struct {
		Results []struct {
			ID           int64    `json:"id"`
			Status       string   `json:"status"`
			DateCreated  string   `json:"date_created"`
			ShippingCost *float64 `json:"shipping_cost"` // pode ser nulo
			OrderItems   []struct {
				Item struct {
					ID            string `json:"id"`
					CategoryID    string `json:"category_id"`
					ListingTypeID string `json:"listing_type_id"`
				} `json:"item"`
				Quantity  int     `json:"quantity"`
				UnitPrice float64 `json:"unit_price"`
				SaleFee   float64 `json:"sale_fee"`
			} `json:"order_items"`
		} `json:"results"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	var orders []Order
	for _, r := range raw.Results {
		order := Order{
			OrderID:      r.ID,
			Status:       r.Status,
			DateCreated:  r.DateCreated,
			ShippingCost: 0,
		}

		if r.ShippingCost != nil {
			order.ShippingCost = *r.ShippingCost
		}

		for _, oi := range r.OrderItems {
			item := OrderItem{
				ItemID:        oi.Item.ID,
				CategoryID:    oi.Item.CategoryID,
				Quantity:      oi.Quantity,
				UnitPrice:     oi.UnitPrice,
				ListingTypeID: oi.Item.ListingTypeID,
				SaleFee:       oi.SaleFee,
			}
			order.Items = append(order.Items, item)
		}

		orders = append(orders, order)
	}

	return orders, nil
}
