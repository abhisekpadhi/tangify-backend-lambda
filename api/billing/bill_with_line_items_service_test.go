package billing

import "testing"

func TestInvoiceWorkerBillIDIsStableForCheckoutState(t *testing.T) {
	t.Parallel()

	first := invoiceWorkerBillID(" order-session:ord-123::checkout ")
	second := invoiceWorkerBillID("order-session:ord-123::checkout")

	if first != second {
		t.Fatalf("expected stable worker id, got %q and %q", first, second)
	}
	if first == invoiceWorkerBillID("order-session:ord-456::checkout") {
		t.Fatal("different checkout states must not share a worker id")
	}
}
