package shipments

import (
	"dimi/kkalcs/mlapi/requests"
	"encoding/json"
	"fmt"
)

type ShipmentCost struct {
	ShipmentID string  // Vamos precisar injetar ou passar isso externamente
	Cost       float64 // Quanto o vendedor pagou de frete
	ChargeFlex float64 // Taxa adicional se usada entrega Flex
	Discount   float64
	FinalCost  float64 // o valor realmente pago pelo vendedor
}

func FetchCosts(shipmentID string) (*ShipmentCost, error) {
	url := fmt.Sprintf("https://api.mercadolibre.com/shipments/%s/costs", shipmentID)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	//fmt.Println(string(body))

	shipmentCost, err := extract(body, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair dados: %s", err)
	}

	return shipmentCost, nil
}

func Fetch(shipmentID string) (int, error) {
	url := fmt.Sprintf("https://api.mercadolibre.com/shipments/%s", shipmentID)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return 0, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	fmt.Println("FECHING RESULT:", string(body))

	var raw struct {
		Order_id int `json:"order_id"`
	}
	err = json.Unmarshal(body, &raw)
	if err != nil {
		return 0, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	return raw.Order_id, nil
}

func extract(data []byte, shipmentID string) (*ShipmentCost, error) {
	var raw struct {
		Senders []struct {
			Cost    float64 `json:"cost"`
			Charges struct {
				ChargeFlex float64 `json:"charge_flex"`
			} `json:"charges"`
			Discounts []struct {
				Rate           float64 `json:"rate"`
				Type           string  `json:"type"`
				PromotedAmount float64 `json:"promoted_amount"`
			} `json:"discounts"`
		} `json:"senders"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	if len(raw.Senders) == 0 {
		return nil, fmt.Errorf("nenhum vendedor encontrado no shipment")
	}

	s := raw.Senders[0]

	// Somar todos os descontos aplicados
	var totalDiscount float64
	for _, d := range s.Discounts {
		totalDiscount += d.PromotedAmount
	}

	// Custo real pago pelo vendedor = custo - descontos
	if (s.Cost - totalDiscount) < 0 {
		fmt.Println("Desconto maior que o custo", s.Cost-totalDiscount, "ID: ", shipmentID)
	}

	finalCost := max(s.Cost-totalDiscount, 0)

	result := &ShipmentCost{
		ShipmentID: shipmentID,
		Cost:       s.Cost,
		ChargeFlex: s.Charges.ChargeFlex,
		Discount:   totalDiscount,
		FinalCost:  finalCost,
	}

	return result, nil
}

func Total(all_shipments []ShipmentCost) float64 {
	var total float64
	for _, s := range all_shipments {
		total += s.FinalCost
	}

	fmt.Println("Total de frete:", total)

	return total
}
