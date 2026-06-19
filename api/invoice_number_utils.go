package main

import (
	"context"

	"tangify-backend-lambda/billing"
)

func fetchInvoiceNumber(ctx context.Context, billID string) (*invoiceNumberWorkerResponse, error) {
	resp, err := billing.FetchInvoiceNumber(ctx, billID)
	if err != nil {
		return nil, err
	}
	return &invoiceNumberWorkerResponse{
		InvoiceNumber: resp.InvoiceNumber,
		BillID:        resp.BillID,
		Year:          resp.Year,
		Sequence:      resp.Sequence,
	}, nil
}

type invoiceNumberWorkerResponse struct {
	InvoiceNumber string `json:"invoice_number"`
	BillID        string `json:"bill_id"`
	Year          int    `json:"year"`
	Sequence      int    `json:"sequence"`
}
