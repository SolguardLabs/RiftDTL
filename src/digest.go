package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func DigestString(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))[:32]
}

func DigestJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return DigestString("json-error", err.Error())
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])[:32]
}

func DigestMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		parts = append(parts, key, values[key])
	}
	return DigestString(parts...)
}
