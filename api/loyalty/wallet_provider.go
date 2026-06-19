package loyalty

import "context"

// WalletProvider implements billing.PointsWalletProvider.
type WalletProvider struct {
	repo *Repository
}

func NewWalletProvider(repo *Repository) *WalletProvider {
	return &WalletProvider{repo: repo}
}

func (p *WalletProvider) GetPointsBalance(ctx context.Context, userID string) (int64, error) {
	w, err := p.repo.GetWallet(ctx, userID)
	if err != nil {
		return 0, err
	}
	return w.PointsBalance, nil
}
