package main

type Event struct {
	ID       string         `json:"id"`
	Clock    int64          `json:"clock"`
	Kind     string         `json:"kind"`
	Subject  string         `json:"subject"`
	ShardID  string         `json:"shard_id,omitempty"`
	Account  string         `json:"account,omitempty"`
	Asset    string         `json:"asset,omitempty"`
	Amount   Amount         `json:"amount,omitempty"`
	Transfer string         `json:"transfer,omitempty"`
	Window   string         `json:"window,omitempty"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

type Journal struct {
	seq    *Sequence
	events []Event
}

func NewJournal() *Journal {
	return &Journal{seq: NewSequence("evt"), events: []Event{}}
}

func (j *Journal) Add(clock int64, kind string, subject string, attrs map[string]any) Event {
	event := Event{
		ID:      j.seq.ID(),
		Clock:   clock,
		Kind:    kind,
		Subject: subject,
		Attrs:   attrs,
	}
	if attrs != nil {
		if shard, ok := attrs["shard_id"].(string); ok {
			event.ShardID = shard
		}
		if account, ok := attrs["account"].(string); ok {
			event.Account = account
		}
		if asset, ok := attrs["asset"].(string); ok {
			event.Asset = asset
		}
		if amount, ok := attrs["amount"].(Amount); ok {
			event.Amount = amount
		}
		if transfer, ok := attrs["transfer"].(string); ok {
			event.Transfer = transfer
		}
		if window, ok := attrs["window"].(string); ok {
			event.Window = window
		}
	}
	j.events = append(j.events, event)
	return event
}

func (j *Journal) List() []Event {
	out := make([]Event, len(j.events))
	copy(out, j.events)
	return out
}

func (j *Journal) Count(kind string) int {
	total := 0
	for _, event := range j.events {
		if event.Kind == kind {
			total++
		}
	}
	return total
}

func (j *Journal) Filter(kind string) []Event {
	out := []Event{}
	for _, event := range j.events {
		if event.Kind == kind {
			out = append(out, event)
		}
	}
	return out
}
