package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const invoiceNumberWorkerURL = "https://invoice-number-cf-worker.subnub.workers.dev/"

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

// FetchInvoiceNumber allocates a sequential invoice number from the worker.
func FetchInvoiceNumber(ctx context.Context, billID string) (*InvoiceNumberResponse, error) {
	reqBody := invoiceNumberWorkerRequest{BillID: billID}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, invoiceNumberWorkerURL, bytes.NewReader(b))
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
