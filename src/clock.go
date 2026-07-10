package main

type Clock struct {
	Now int64 `json:"now"`
}

func NewClock() *Clock {
	return &Clock{Now: 0}
}

func (c *Clock) Tick() int64 {
	c.Now++
	return c.Now
}

func (c *Clock) Advance(delta int64) int64 {
	if delta <= 0 {
		return c.Now
	}
	c.Now += delta
	return c.Now
}

func (c *Clock) Snapshot() int64 {
	return c.Now
}

type Clocked interface {
	SetClock(clock int64)
}

func Stamp(clock int64, value Clocked) {
	if value != nil {
		value.SetClock(clock)
	}
}
