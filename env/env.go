package env

import (
	"fmt"
	"os"
)

func FrontendPort() string {
	return fmt.Sprintf(
		":%s",
		lookupWithFallback("FRONTEND_PORT", "8008"),
	)
}

func ConfigPath() string {
	return lookupWithFallback("CONFIG_PATH", "/app/kvs")
}

func DBHost() string {
	return lookupWithFallback("SQUEAL_HOST", "squeal")
}

func DBUser() string {
	return lookupWithFallback("SQUEAL_USER", "test")
}

func DBPass() string {
	return lookupWithFallback("SQUEAL_PASS", "verySecureSuperSafe")
}

func DBName() string {
	return lookupWithFallback("SQUEAL_DB", "cloudKV")
}

func lookupWithFallback(key, fallback string) string {
	value, found := os.LookupEnv(key)
	if found {
		return value
	}
	if fallback != "" {
		return fallback
	}

	return ""
}
