package payments

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"dimi/kkalcs/shpeapi/requests" // Reutilizando seu pacote de requisições
)

const (
	shopeeBaseURL       = "https://partner.shopeemobile.com"
	escrowDetailAPIPath = "/api/v2/payment/get_escrow_detail_batch" // CORRIGIDO
	maxOrdersPerBatch   = 50
)

// EscrowDetail captura os dados financeiros essenciais para o cálculo de lucro
// e reconciliação, focando no valor final fornecido pela API.
type EscrowDetail struct {
	OrderSN string `json:"order_sn"`

	// --- CAMPO FUNDAMENTAL ---
	// O resultado final creditado pela Shopee. Use este como sua fonte da verdade.
	EscrowAmount float64 `json:"escrow_amount"`

	// --- COMPONENTES PARA RECONCILIAÇÃO E CÁLCULOS SECUNDÁRIOS ---

	// Custos descontados do vendedor (principais)
	CommissionFee        float64 `json:"commission_fee"`
	ServiceFee           float64 `json:"service_fee"`
	SellerTransactionFee float64 `json:"seller_transaction_fee"`
	FinalShippingFee     float64 `json:"final_shipping_fee"`
	ActualShippingFee    float64 `json:"actual_shipping_fee"`  // Custo total do frete
	ReverseShippingFee   float64 `json:"reverse_shipping_fee"` // Custo do frete reverso

	// Subsídios fornecidos pela Shopee (para calcular a "ajuda" da plataforma)
	ShopeeDiscount       float64 `json:"shopee_discount"`
	VoucherFromShopee    float64 `json:"voucher_from_shopee"`
	ShippingFeeDiscount  float64 `json:"shipping_fee_discount_from_shopee"` // Subsídio de frete
	ShopeeShippingRebate float64 `json:"shopee_shipping_rebate"`            // Tag JSON corrigida

	VoucherFromSeller  float64 `json:"voucher_from_seller"`
	CoinsAmount        float64 `json:"coins_amount"`
	BundleDealDiscount float64 `json:"bundle_deal_discount"`

	// Valor de referência bruto
	OriginalPrice float64 `json:"original_price"`
}

type APIResponse struct {
	Error     string          `json:"error"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
	Response  []EscrowWrapper `json:"response"` // CORREÇÃO: 'response' é um array.
}

// EscrowWrapper corresponde a cada elemento no array 'response'.
type EscrowWrapper struct {
	Detail FullEscrowDetail `json:"escrow_detail"`
}

// FullEscrowDetail corresponde à estrutura aninhada 'escrow_detail'.
type FullEscrowDetail struct {
	OrderSN     string      `json:"order_sn"`
	OrderIncome OrderIncome `json:"order_income"`
}

// OrderIncome corresponde à estrutura 'order_income' com os dados financeiros.
type OrderIncome struct {
	EscrowAmount         float64 `json:"escrow_amount"`
	CommissionFee        float64 `json:"commission_fee"`
	ServiceFee           float64 `json:"service_fee"`
	FinalShippingFee     float64 `json:"final_shipping_fee"`
	ReverseShippingFee   float64 `json:"reverse_shipping_fee"`
	ShopeeDiscount       float64 `json:"shopee_discount"`
	VoucherFromShopee    float64 `json:"voucher_from_shopee"`
	ShopeeShippingRebate float64 `json:"shopee_shipping_rebate"`
	OriginalPrice        float64 `json:"original_price"`
	VoucherFromSeller    float64 `json:"voucher_from_seller"`
	CoinsAmount          float64 `json:"coins_amount"`
	BundleDealDiscount   float64 `json:"bundle_deal_discount"`
}

// GetEscrowDetailBatch busca os detalhes financeiros para uma lista de pedidos.
// A função foi CORRIGIDA para usar o método POST e enviar a lista no corpo da requisição.
func GetEscrowDetailBatch(orderSNs []string) (map[string]EscrowDetail, error) {
	if len(orderSNs) == 0 {
		return nil, fmt.Errorf("a lista de order_sn não pode estar vazia")
	}

	allDetails := make(map[string]EscrowDetail)

	for i := 0; i < len(orderSNs); i += maxOrdersPerBatch {
		end := i + maxOrdersPerBatch
		if end > len(orderSNs) {
			end = len(orderSNs)
		}
		batch := orderSNs[i:end]

		slog.Info("Fetching escrow details for batch", "batch_number", (i/maxOrdersPerBatch)+1, "batch_size", len(batch))

		requestBody := map[string][]string{
			"order_sn_list": batch,
		}

		// 2. Chama MakeShopeeRequest com o método POST, sem queryParams na URL, e com o corpo da requisição.
		responseBody, err := requests.MakeShopeeRequest(requests.POST, shopeeBaseURL, escrowDetailAPIPath, nil, requestBody)
		if err != nil {
			return nil, fmt.Errorf("requisição para get_escrow_detail falhou para o lote a partir do índice %d: %w", i, err)
		}

		var apiResponse APIResponse

		if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
			slog.Error("Falha ao decodificar a resposta da API", "error", err, "raw_response", string(responseBody))
			return nil, fmt.Errorf("falha ao decodificar a resposta do lote a partir do índice %d: %w", i, err)
		}

		if apiResponse.Error != "" {
			// Adiciona o corpo da resposta no log de erro para depuração completa.
			slog.Error("Erro da API Shopee", "error", apiResponse.Error, "message", apiResponse.Message, "raw_response", string(responseBody))
			return nil, fmt.Errorf("erro da API Shopee no lote a partir do índice %d: %s - %s", i, apiResponse.Error, apiResponse.Message)
		}

		for _, wrapper := range apiResponse.Response {
			income := wrapper.Detail.OrderIncome
			detail := EscrowDetail{
				OrderSN:              wrapper.Detail.OrderSN,
				EscrowAmount:         income.EscrowAmount,
				CommissionFee:        income.CommissionFee,
				ServiceFee:           income.ServiceFee,
				FinalShippingFee:     income.FinalShippingFee,
				ReverseShippingFee:   income.ReverseShippingFee,
				ShopeeDiscount:       income.ShopeeDiscount,
				VoucherFromShopee:    income.VoucherFromShopee,
				ShopeeShippingRebate: income.ShopeeShippingRebate,
				OriginalPrice:        income.OriginalPrice,
				VoucherFromSeller:    income.VoucherFromSeller,
				CoinsAmount:          income.CoinsAmount,
				BundleDealDiscount:   income.BundleDealDiscount,
			}
			allDetails[detail.OrderSN] = detail
		}
	}

	slog.Info("Finished fetching all escrow details.", "total_orders_processed", len(allDetails))
	return allDetails, nil
}
