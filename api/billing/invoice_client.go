package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultInvoiceNumberWorkerURL = "https://invoice-number-cf-worker.subnub.workers.dev/"
const invoiceNumberWorkerURLEnvProd = "INVOICE_NUMBER_WORKER_URL_PROD"
const invoiceNumberWorkerURLEnvDev = "INVOICE_NUMBER_WORKER_URL_DEV"

type invoiceNumberWorkerRequest struct {
	BillID string `json:"bill_id"`
}

// InvoiceNumberResponse is returned by the invoice number worker.
type InvoiceNumberResponse struct {
	InvoiceNumber string `json:"invoice_number"`
	BillID        string `json:"bill_id"`
	Year          int    `json:"year"`
	Sequence      int    `json:"sequence"`
}

func ResolveInvoiceWorkerURL(environment string) string {
	if strings.EqualFold(strings.TrimSpace(environment), "dev") {
		if v := strings.TrimSpace(os.Getenv(invoiceNumberWorkerURLEnvDev)); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(os.Getenv(invoiceNumberWorkerURLEnvProd)); v != "" {
		return v
	}
	return defaultInvoiceNumberWorkerURL
}

// FetchInvoiceNumber allocates a sequential invoice number from the worker.
func FetchInvoiceNumber(ctx context.Context, billID string) (*InvoiceNumberResponse, error) {
	return FetchInvoiceNumberWithURL(ctx, billID, ResolveInvoiceWorkerURL("production"))
}

// FetchInvoiceNumberWithURL allocates a sequential invoice number from the worker URL.
func FetchInvoiceNumberWithURL(
	ctx context.Context,
	billID string,
	workerURL string,
) (*InvoiceNumberResponse, error) {
	targetURL := strings.TrimSpace(workerURL)
	if targetURL == "" {
		targetURL = defaultInvoiceNumberWorkerURL
	}

	reqBody := invoiceNumberWorkerRequest{BillID: billID}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		targetURL,
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invoice worker status=%d body=%s", resp.StatusCode, string(respBody))
	}

	out := &InvoiceNumberResponse{}
	if err := json.Unmarshal(respBody, out); err != nil {
		return nil, err
	}
	if out.InvoiceNumber == "" {
		return nil, fmt.Errorf("invoice worker returned empty invoice_number")
	}
	return out, nil
}
