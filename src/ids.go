package main

import (
	"fmt"
	"sort"
	"strings"
)

type Sequence struct {
	Prefix string `json:"prefix"`
	Next   int64  `json:"next"`
}

func NewSequence(prefix string) *Sequence {
	return &Sequence{Prefix: prefix, Next: 1}
}

func (s *Sequence) ID() string {
	value := fmt.Sprintf("%s-%04d", s.Prefix, s.Next)
	s.Next++
	return value
}

func NormalizeID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func RequireID(kind string, value string) (string, error) {
	id := NormalizeID(value)
	if id == "" {
		return "", fmt.Errorf("%s id required", kind)
	}
	return id, nil
}

func JoinID(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if normalized := NormalizeID(part); normalized != "" {
			clean = append(clean, normalized)
		}
	}
	return strings.Join(clean, "-")
}

func TransferKey(sourceShard string, destShard string, asset string, nonce int64) string {
	return fmt.Sprintf("xfer-%s-%s-%s-%04d", NormalizeID(sourceShard), NormalizeID(destShard), NormalizeID(asset), nonce)
}

func WithdrawalKey(shardID string, accountID string, asset string, nonce int64) string {
	return fmt.Sprintf("wd-%s-%s-%s-%04d", NormalizeID(shardID), NormalizeID(accountID), NormalizeID(asset), nonce)
}

func ClosureKey(shardID string, clock int64) string {
	return fmt.Sprintf("close-%s-%06d", NormalizeID(shardID), clock)
}

func CreditGroupKey(destShard string, accountID string, asset string) string {
	return strings.Join([]string{NormalizeID(destShard), NormalizeID(accountID), NormalizeID(asset)}, ":")
}

func DebitQueueKey(sourceShard string, transferID string) string {
	return strings.Join([]string{NormalizeID(sourceShard), NormalizeID(transferID)}, ":")
}

func ContainsID(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func UniqueIDs(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		id := NormalizeID(value)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func SortStrings(values []string) {
	sort.Strings(values)
}

func CloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func AppendUnique(values []string, next string) []string {
	next = NormalizeID(next)
	if next == "" || ContainsID(values, next) {
		return values
	}
	return append(values, next)
}

func MustID(kind string, value string) string {
	id, err := RequireID(kind, value)
	if err != nil {
		panic(err)
	}
	return id
}
