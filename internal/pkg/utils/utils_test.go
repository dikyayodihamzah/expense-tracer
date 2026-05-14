package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormatNominal(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1.000"},
		{10000, "10.000"},
		{100000, "100.000"},
		{1000000, "1.000.000"},
		{1500000, "1.500.000"},
		{123456789, "123.456.789"},
		{-1000, "-1.000"},
		{-1500000, "-1.500.000"},
	}
	for _, tt := range tests {
		if got := FormatNominal(tt.input); got != tt.want {
			t.Errorf("FormatNominal(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMustEnv(t *testing.T) {
	t.Setenv("TEST_MUSTENV_KEY", "hello")
	if got := MustEnv("TEST_MUSTENV_KEY"); got != "hello" {
		t.Errorf("MustEnv = %q, want %q", got, "hello")
	}
}

func TestDownloadFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("receipt data"))
	}))
	defer srv.Close()

	data, err := DownloadFile(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "receipt data" {
		t.Errorf("got %q, want %q", data, "receipt data")
	}
}

func TestDownloadFile_BadURL(t *testing.T) {
	_, err := DownloadFile("http://127.0.0.1:0/no-server")
	if err == nil {
		t.Error("expected error for unreachable URL")
	}
}

func TestFormatNominalFromStr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1,000", "1.000"},
		{"1,000,000", "1.000.000"},
		{"100", "100"},
		{"", ""},
		{"1,500,000", "1.500.000"},
	}
	for _, tt := range tests {
		if got := FormatNominalFromStr(tt.input); got != tt.want {
			t.Errorf("FormatNominalFromStr(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
