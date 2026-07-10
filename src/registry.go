package main

import "fmt"

type ShardStore struct {
	order []string
	items map[string]*Shard
}

func NewShardStore() *ShardStore {
	return &ShardStore{order: []string{}, items: map[string]*Shard{}}
}

func (s *ShardStore) Add(shard *Shard) error {
	if shard == nil {
		return fmt.Errorf("nil shard")
	}
	if _, ok := s.items[shard.ID]; !ok {
		s.order = append(s.order, shard.ID)
	}
	s.items[shard.ID] = shard
	return nil
}

func (s *ShardStore) MustAdd(shard *Shard) {
	if err := s.Add(shard); err != nil {
		panic(err)
	}
}

func (s *ShardStore) Get(id string) (*Shard, bool) {
	shard, ok := s.items[NormalizeID(id)]
	return shard, ok
}

func (s *ShardStore) MustGet(id string) *Shard {
	shard, ok := s.Get(id)
	if !ok {
		panic(fmt.Sprintf("missing shard %s", id))
	}
	return shard
}

func (s *ShardStore) List() []*Shard {
	out := make([]*Shard, 0, len(s.order))
	for _, id := range s.order {
		if shard, ok := s.items[id]; ok {
			out = append(out, shard)
		}
	}
	return out
}

func (s *ShardStore) Reports() []ShardReport {
	shards := s.List()
	out := make([]ShardReport, 0, len(shards))
	for _, shard := range shards {
		out = append(out, shard.Report())
	}
	return out
}

func (s *ShardStore) IDs() []string {
	return CloneStrings(s.order)
}

func (s *ShardStore) Accounts() []AccountReport {
	out := []AccountReport{}
	for _, shard := range s.List() {
		out = append(out, shard.Accounts.Reports()...)
	}
	return out
}

func (s *ShardStore) TotalAvailable(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Available(asset)
	}
	return total
}

func (s *ShardStore) TotalHeld(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Held(asset)
	}
	return total
}

func (s *ShardStore) TotalPendingIn(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Accounts.TotalPendingIn(asset)
	}
	return total
}

func (s *ShardStore) TotalPendingOut(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Accounts.TotalPendingOut(asset)
	}
	return total
}

func (s *ShardStore) TotalWithdrawn(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Withdrawn(asset)
	}
	return total
}

func (s *ShardStore) TotalExternalized(asset string) Amount {
	total := Amount(0)
	for _, shard := range s.List() {
		total += shard.Externalized(asset)
	}
	return total
}
