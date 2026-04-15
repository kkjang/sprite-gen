package secrets

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func Load(keys ...string) (string, error) {
	fileValues := map[string]string{}
	if values, err := readEnvFile(".env"); err != nil {
		return "", err
	} else {
		for k, v := range values {
			fileValues[k] = v
		}
	}
	if values, err := readEnvFile(".env.local"); err != nil {
		return "", err
	} else {
		for k, v := range values {
			fileValues[k] = v
		}
	}

	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok && value != "" {
			return value, nil
		}
		if value := fileValues[key]; value != "" {
			return value, nil
		}
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("no secret keys requested")
	}
	return "", fmt.Errorf("missing secret; set one of %s", strings.Join(keys, ", "))
}

func Redact(s, key string) string {
	if key == "" {
		return s
	}
	return strings.ReplaceAll(s, key, "***REDACTED***")
}

func readEnvFile(path string) (map[string]string, error) {
	values, err := godotenv.Read(path)
	if err == nil {
		return values, nil
	}
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	return nil, fmt.Errorf("read %s: %w", path, err)
}
