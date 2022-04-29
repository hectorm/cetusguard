package env

import (
	"os"
	"strconv"
)

func StringEnv(def string, keys ...string) string {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
	}
	return def
}

func StringSliceEnv(def []string, keys ...string) []string {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			return []string{val}
		}
	}
	return def
}

func IntEnv(def int, keys ...string) int {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		}
	}
	return def
}

func BoolEnv(def bool, keys ...string) bool {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			if b, err := strconv.ParseBool(val); err == nil {
				return b
			}
		}
	}
	return def
}
