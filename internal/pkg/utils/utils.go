package utils

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// FormatNominal formats an integer using Indonesian number format (dot as thousand separator).
// Negative values are prefixed with a minus sign.
func FormatNominal(n int64) string {
	if n < 0 {
		return "-" + FormatNominal(-n)
	}
	s := strconv.FormatInt(n, 10)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// FormatNominalFromStr formats a string using Indonesian number format (dot as thousand separator).
func FormatNominalFromStr(s string) string {
	return strings.ReplaceAll(s, ",", ".")
}

// MustEnv returns the value of an environment variable or fatals if it is empty.
func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}

// DownloadFile fetches a URL and returns the body, capped at 20 MB.
func DownloadFile(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
}
