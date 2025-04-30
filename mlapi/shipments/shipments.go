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
}

func FetchCosts(shipmentID string) (*ShipmentCost, error) {
	url := fmt.Sprintf("https://api.mercadolibre.com/shipments/%s/costs", shipmentID)

	body, err := requests.MakeSimpleRequest(requests.GET, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %s", err)
	}

	shipmentCost, err := extract(body, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("erro ao extrair dados: %s", err)
	}

	return shipmentCost, nil
}

func extract(data []byte, shipmentID string) (*ShipmentCost, error) {
	var raw struct {
		Senders []struct {
			Cost    float64 `json:"cost"`
			Charges struct {
				ChargeFlex float64 `json:"charge_flex"`
			} `json:"charges"`
		} `json:"senders"`
	}

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer unmarshal: %v", err)
	}

	if len(raw.Senders) == 0 {
		return nil, fmt.Errorf("nenhum vendedor encontrado no shipment")
	}

	// Normalmente só existe um sender (o vendedor principal)
	s := raw.Senders[0]

	result := &ShipmentCost{
		ShipmentID: shipmentID,
		Cost:       s.Cost,
		ChargeFlex: s.Charges.ChargeFlex,
	}

	return result, nil
}
