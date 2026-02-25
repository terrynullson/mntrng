package config

import "testing"

func TestIsProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	if !IsProduction() {
		t.Fatalf("expected production env")
	}
}

func TestValidateAPIRuntimeSafetyRejectsInsecureProductionConfig(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_COOKIE_SECURE", "false")

	err := ValidateAPIRuntimeSafety()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAPIRuntimeSafetyRejectsPublicMetricsInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("API_METRICS_PUBLIC", "true")

	err := ValidateAPIRuntimeSafety()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAPIRuntimeSafetyAllowsLocalHTTPOriginsForDevProxy(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")

	if err := ValidateAPIRuntimeSafety(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAPIRuntimeSafetyRejectsNonTLSOriginsInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://app.example.com")

	err := ValidateAPIRuntimeSafety()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAPIRuntimeSafetyNoopInNonProduction(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	t.Setenv("API_METRICS_PUBLIC", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://app.example.com")

	if err := ValidateAPIRuntimeSafety(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
