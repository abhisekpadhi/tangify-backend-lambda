package loyalty

import "testing"

func TestRepositoryTableSelection(t *testing.T) {
	t.Parallel()

	production := NewRepository(nil)
	if production.tableName != TableNamePointsWallet {
		t.Fatalf("expected production table, got %q", production.tableName)
	}

	development := NewRepository(nil, DevTableNamePointsWallet)
	if development.tableName != DevTableNamePointsWallet {
		t.Fatalf("expected development table, got %q", development.tableName)
	}
}
