package main

import (
	"fmt"
	"math"
	"strconv"
)

type Amount int64

const (
	ZeroAmount Amount = 0
	MaxAmount  Amount = Amount(math.MaxInt64)
)

type AmountBucket struct {
	Asset  string `json:"asset"`
	Amount Amount `json:"amount"`
}

func NewAmount(value int64) (Amount, error) {
	if value < 0 {
		return 0, fmt.Errorf("negative amount %d", value)
	}
	return Amount(value), nil
}

func MustAmount(value int64) Amount {
	amount, err := NewAmount(value)
	if err != nil {
		panic(err)
	}
	return amount
}

func Bucket(asset string, amount Amount) AmountBucket {
	return AmountBucket{Asset: asset, Amount: amount}
}

func (a Amount) Int64() int64 {
	return int64(a)
}

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) IsZero() bool {
	return a == 0
}

func (a Amount) Positive() bool {
	return a > 0
}

func (a Amount) Negative() bool {
	return a < 0
}

func (a Amount) Add(b Amount) Amount {
	if b > 0 && a > MaxAmount-b {
		return MaxAmount
	}
	return a + b
}

func (a Amount) Sub(b Amount) Amount {
	return a - b
}

func (a Amount) ClampZero() Amount {
	if a < 0 {
		return 0
	}
	return a
}

func (a Amount) Min(b Amount) Amount {
	if a < b {
		return a
	}
	return b
}

func (a Amount) Max(b Amount) Amount {
	if a > b {
		return a
	}
	return b
}

func (a Amount) MulInt(v int64) Amount {
	if v <= 0 || a <= 0 {
		return 0
	}
	if int64(a) > math.MaxInt64/v {
		return MaxAmount
	}
	return Amount(int64(a) * v)
}

func (a Amount) DivInt(v int64) Amount {
	if v <= 0 {
		return 0
	}
	return Amount(int64(a) / v)
}

func (a Amount) Bps(bps int) Amount {
	if bps <= 0 || a <= 0 {
		return 0
	}
	return Amount((int64(a) * int64(bps)) / 10_000)
}

func (a Amount) Percent(percent int) Amount {
	return a.Bps(percent * 100)
}

func (a Amount) Share(numerator Amount, denominator Amount) Amount {
	if denominator <= 0 || numerator <= 0 || a <= 0 {
		return 0
	}
	return Amount((int64(a) * int64(numerator)) / int64(denominator))
}

func (a Amount) Compare(b Amount) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func RequireAmount(label string, amount Amount) error {
	if amount <= 0 {
		return fmt.Errorf("%s must be positive", label)
	}
	return nil
}

func RequireAvailable(label string, available Amount, requested Amount) error {
	if requested <= 0 {
		return fmt.Errorf("%s requested amount must be positive", label)
	}
	if available < requested {
		return fmt.Errorf("%s insufficient funds: available=%d requested=%d", label, available, requested)
	}
	return nil
}

func SumAmounts(values []Amount) Amount {
	total := Amount(0)
	for _, value := range values {
		total += value
	}
	return total
}

func SumBuckets(buckets []AmountBucket, asset string) Amount {
	total := Amount(0)
	for _, bucket := range buckets {
		if bucket.Asset == asset {
			total += bucket.Amount
		}
	}
	return total
}

func BucketMap(buckets []AmountBucket) map[string]Amount {
	out := map[string]Amount{}
	for _, bucket := range buckets {
		out[bucket.Asset] += bucket.Amount
	}
	return out
}

func BucketsFromMap(values map[string]Amount) []AmountBucket {
	keys := SortedAmountKeys(values)
	out := make([]AmountBucket, 0, len(keys))
	for _, key := range keys {
		out = append(out, Bucket(key, values[key]))
	}
	return out
}

func SortedAmountKeys(values map[string]Amount) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	SortStrings(keys)
	return keys
}

func AddBucket(values map[string]Amount, asset string, amount Amount) {
	if asset == "" || amount == 0 {
		return
	}
	values[asset] += amount
}

func SubBucket(values map[string]Amount, asset string, amount Amount) {
	if asset == "" || amount == 0 {
		return
	}
	values[asset] -= amount
}
