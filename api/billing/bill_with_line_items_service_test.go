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

func TestBillRepositoryTableSelection(t *testing.T) {
	t.Parallel()

	production := NewBillWithLineItemsRepository(nil)
	if production.tableName != TableNameBillsWithLineItems {
		t.Fatalf("expected production table, got %q", production.tableName)
	}

	development := NewBillWithLineItemsRepository(
		nil,
		DevTableNameBillsWithLineItems,
	)
	if development.tableName != DevTableNameBillsWithLineItems {
		t.Fatalf("expected development table, got %q", development.tableName)
	}
}
