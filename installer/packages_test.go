package main

import "testing"

func TestPrometheusValidateAlertSignature(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		expected  string
	}{
		{name: "custom signature is preserved", signature: "prod-cluster", expected: "prod-cluster"},
		{name: "surrounding whitespace is trimmed", signature: "  prod-cluster  ", expected: "prod-cluster"},
		{name: "empty falls back to default", signature: "", expected: "monitoring"},
		{name: "whitespace falls back to default", signature: "   ", expected: "monitoring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := PrometheusConfig{
				alertmanagerStorageSizeGi: 1,
				grafanaStorageSizeGi:      1,
				prometheusStorageSizeGi:   1,
				storageClass:              "standard",
				grafanaRootUrl:            "https://example.com/grafana",
				adapterSignature:          tt.signature,
			}

			if err := config.validate(); err != nil {
				t.Fatalf("validate() returned error: %v", err)
			}
			if config.adapterSignature != tt.expected {
				t.Fatalf("adapterSignature = %q, want %q", config.adapterSignature, tt.expected)
			}
		})
	}
}
