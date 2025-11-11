package eth

import (
	"os"
	"strconv"
)

// lookupEnv is a tiny indirection to avoid importing os in this file's API surface.
var lookupEnv = func(key string) string { return getEnv(key) }

func getenvDefault(key, def string) string {
	if v := lookupEnv(key); v != "" {
		return v
	}
	return def
}

func getenvInt64Default(key string, def int64) int64 {
	if v := lookupEnv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func getEnv(key string) string { return os.Getenv(key) }
