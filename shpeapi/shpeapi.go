package shpeapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"dimi/kkalcs/shpeapi/orders"
	"dimi/kkalcs/shpeapi/payments"
)

// ReconciledOrder combina os dados do pedido com os dados financeiros.
type ReconciledOrder struct {
	OrderData      *orders.ShopeeOrder   `json:"order_data,omitempty"`
	FinancialsData payments.EscrowDetail `json:"financials_data"`
}

// MetricsResult armazena os resultados financeiros agregados.
// Os nomes foram ajustados para maior clareza.
type MetricsResult struct {
	// TotalNetCredited é a soma de todos os 'EscrowAmount', o valor real que entrou na sua conta.
	TotalNetCredited float64
	// TotalGrossRevenue é a soma do valor de venda bruto (preço original dos produtos).
	TotalGrossRevenue float64
	// TotalPlatformFees é a soma das taxas cobradas pela plataforma.
	TotalPlatformFees float64
	// TotalShippingCost é a soma dos custos de frete (real - subsídio).
	TotalShippingCost float64
	// TotalShopeeSubsidies é a soma de todos os descontos e subsídios bancados pela Shopee.
	TotalShopeeSubsidies float64
	TransactionCount     int
	TotalSellerDiscounts float64 // Soma de todos os descontos dados pelo vendedor (vouchers, moedas, etc.)
}

// FetchAndReconcileTransactions orquestra o fluxo de busca e reconciliação.
// Agora lida com o limite de 15 dias da API dividindo o período em chunks.
func FetchAndReconcileTransactions(dateFrom, dateTo time.Time) ([]ReconciledOrder, error) {
	slog.Info("Step 1: Fetching order SNs for the given date range in 15-day chunks...")

	var allOrderSNs []string

	// Split the date range into 15-day chunks
	for currentFrom := dateFrom; currentFrom.Before(dateTo); {
		// Calculate the end date for this chunk (max 15 days)
		currentTo := currentFrom.AddDate(0, 0, 15)
		if currentTo.After(dateTo) {
			currentTo = dateTo
		}

		slog.Info("Fetching orders for chunk",
			"from", currentFrom.Format("2006-01-02"),
			"to", currentTo.Format("2006-01-02"))

		chunkOrderSNs, err := orders.GetOrderListByDateRange(currentFrom, currentTo)
		if err != nil {
			return nil, fmt.Errorf("could not fetch order list for period %s to %s: %w",
				currentFrom.Format("2006-01-02"), currentTo.Format("2006-01-02"), err)
		}

		allOrderSNs = append(allOrderSNs, chunkOrderSNs...)
		slog.Info("Fetched orders for chunk", "count", len(chunkOrderSNs))

		// Move to the next chunk
		currentFrom = currentTo

		// Add a small delay between requests to be respectful to the API
		time.Sleep(100 * time.Millisecond)
	}

	if len(allOrderSNs) == 0 {
		slog.Info("No orders found for the specified period.")
		return []ReconciledOrder{}, nil
	}
	slog.Info("Successfully fetched all order SNs.", "total_count", len(allOrderSNs))

	slog.Info("Step 2: Fetching detailed financial data in batches...")
	financialsMap, err := payments.GetEscrowDetailBatch(allOrderSNs)
	if err != nil {
		return nil, fmt.Errorf("could not fetch escrow details in batch: %w", err)
	}
	slog.Info("Successfully fetched detailed financial data.", "count", len(financialsMap))

	slog.Info("Step 3: Fetching order item details...")
	orderDetailsMap, err := orders.FetchOrderDetailsBySN(allOrderSNs)
	if err != nil {
		slog.Warn("Could not fetch some order details, proceeding with financial data only.", "error", err)
	}

	slog.Info("Step 4: Reconciling all data sources...")
	reconciledOrders := make([]ReconciledOrder, 0, len(financialsMap))
	for sn, financialData := range financialsMap {
		orderDetail, found := orderDetailsMap[sn]

		reco := ReconciledOrder{
			FinancialsData: financialData,
		}
		if found {
			reco.OrderData = &orderDetail
		}

		reconciledOrders = append(reconciledOrders, reco)
	}

	slog.Info("Finished reconciliation process.", "total_reconciled", len(reconciledOrders))
	return reconciledOrders, nil
}

// CalculateDetailedMetrics foi reescrita para usar a lógica correta e mais robusta.
func CalculateDetailedMetrics(reconciledOrders []ReconciledOrder) MetricsResult {
	var metrics MetricsResult
	metrics.TransactionCount = len(reconciledOrders)

	for _, order := range reconciledOrders {
		f := order.FinancialsData // Atalho para os dados financeiros

		// --- CÁLCULO DE LUCRO (A FORMA CORRETA) ---
		// O valor líquido creditado é o 'EscrowAmount'. Esta é a métrica mais importante.
		metrics.TotalNetCredited += f.EscrowAmount

		// --- MÉTRICAS PARA RECONCILIAÇÃO E ANÁLISE ---

		// Receita Bruta (valor total dos produtos antes de qualquer dedução)
		metrics.TotalGrossRevenue += f.OriginalPrice

		// Taxas da Plataforma (custos que a Shopee cobra pelo serviço)
		metrics.TotalPlatformFees += f.CommissionFee + f.ServiceFee + f.SellerTransactionFee

		// Custo Líquido de Frete (o que você efetivamente pagou pelo envio)
		netShippingCost := f.ActualShippingFee - f.ShippingFeeDiscount
		metrics.TotalShippingCost += netShippingCost

		// Subsídios da Shopee (quanto a Shopee "pagou" para ajudar na venda)
		metrics.TotalShopeeSubsidies += f.ShopeeDiscount + f.VoucherFromShopee + f.ShippingFeeDiscount
		totalSellerDiscounts := f.VoucherFromSeller + f.CoinsAmount + f.BundleDealDiscount
		metrics.TotalSellerDiscounts += totalSellerDiscounts
	}

	return metrics
}

// SaveReconciledOrdersToFile salva os dados em um arquivo JSON.
// Nenhuma alteração necessária.
func SaveReconciledOrdersToFile(orders []ReconciledOrder, filename string) error {
	slog.Info("Saving reconciled data to file...", "filename", filename)
	asJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling reconciled orders to JSON: %w", err)
	}
	if err := os.WriteFile(filename, asJSON, 0644); err != nil {
		return fmt.Errorf("error writing file '%s': %w", filename, err)
	}
	slog.Info("Successfully saved data.", "filename", filename)
	return nil
}

// PrintMetricsSummary foi atualizada para exibir as métricas com clareza.
func PrintMetricsSummary(metrics MetricsResult) {
	slog.Info("--- Resumo Financeiro do Período ---")
	slog.Info(fmt.Sprintf("Pedidos processados: %d", metrics.TransactionCount))
	slog.Info("--------------------------------------")
	// A métrica mais importante: o valor que efetivamente entrou na sua conta.
	slog.Info(fmt.Sprintf("  (+) Valor Líquido Creditado (Total Escrow): R$ %.2f", metrics.TotalNetCredited))
	slog.Info("--------------------------------------")
	slog.Info("--- Detalhamento para Reconciliação ---")
	slog.Info(fmt.Sprintf("  - Receita Bruta (Soma dos Preços Originais): R$ %.2f", metrics.TotalGrossRevenue))
	slog.Info(fmt.Sprintf("  - Taxas da Plataforma (Comissão, Serviço, etc.): R$ %.2f", metrics.TotalPlatformFees))
	slog.Info(fmt.Sprintf("  - Custo Líquido de Frete (Pago pelo Vendedor): R$ %.2f", metrics.TotalShippingCost))
	slog.Info(fmt.Sprintf("  - Subsídios da Shopee (Descontos e Frete): R$ %.2f", metrics.TotalShopeeSubsidies))
	slog.Info(fmt.Sprintf("  - Subsídios do Vendedor (Vouchers, Moedas, etc.): R$ %.2f", metrics.TotalSellerDiscounts))
}
