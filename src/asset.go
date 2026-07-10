package main

import "fmt"

type AssetKind string

const (
	AssetStable     AssetKind = "stable"
	AssetUtility    AssetKind = "utility"
	AssetSettlement AssetKind = "settlement"
)

type Asset struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Name          string    `json:"name"`
	Decimals      int       `json:"decimals"`
	Kind          AssetKind `json:"kind"`
	SettlementBps int       `json:"settlement_bps"`
	WithdrawalBps int       `json:"withdrawal_bps"`
	RiskWeightBps int       `json:"risk_weight_bps"`
	Active        bool      `json:"active"`
	Tags          TagSet    `json:"tags"`
}

type AssetStore struct {
	order []string
	items map[string]Asset
}

func NewAsset(id string, symbol string, decimals int, kind AssetKind) Asset {
	return Asset{
		ID:            MustID("asset", id),
		Symbol:        symbol,
		Name:          symbol,
		Decimals:      decimals,
		Kind:          kind,
		SettlementBps: 0,
		WithdrawalBps: 0,
		RiskWeightBps: 10_000,
		Active:        true,
		Tags:          NewTagSet(),
	}
}

func NewAssetStore() *AssetStore {
	return &AssetStore{order: []string{}, items: map[string]Asset{}}
}

func (s *AssetStore) Add(asset Asset) error {
	if asset.ID == "" {
		return fmt.Errorf("asset id required")
	}
	if _, ok := s.items[asset.ID]; !ok {
		s.order = append(s.order, asset.ID)
	}
	if asset.Tags == nil {
		asset.Tags = NewTagSet()
	}
	s.items[asset.ID] = asset
	return nil
}

func (s *AssetStore) MustAdd(asset Asset) {
	if err := s.Add(asset); err != nil {
		panic(err)
	}
}

func (s *AssetStore) Get(id string) (Asset, bool) {
	asset, ok := s.items[NormalizeID(id)]
	return asset, ok
}

func (s *AssetStore) MustGet(id string) Asset {
	asset, ok := s.Get(id)
	if !ok {
		panic(fmt.Sprintf("missing asset %s", id))
	}
	return asset
}

func (s *AssetStore) Has(id string) bool {
	_, ok := s.Get(id)
	return ok
}

func (s *AssetStore) List() []Asset {
	out := make([]Asset, 0, len(s.order))
	for _, id := range s.order {
		if asset, ok := s.items[id]; ok {
			out = append(out, asset)
		}
	}
	return out
}

func (s *AssetStore) IDs() []string {
	return CloneStrings(s.order)
}

func (s *AssetStore) Fee(assetID string, amount Amount, withdrawal bool) Amount {
	asset, ok := s.Get(assetID)
	if !ok {
		return 0
	}
	if withdrawal {
		return amount.Bps(asset.WithdrawalBps)
	}
	return amount.Bps(asset.SettlementBps)
}

func (s *AssetStore) ActiveIDs() []string {
	out := []string{}
	for _, asset := range s.List() {
		if asset.Active {
			out = append(out, asset.ID)
		}
	}
	return out
}
