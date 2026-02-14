package config

import "testing"

func TestGetBoolSupportedValues(t *testing.T) {
	testCases := []struct {
		name     string
		rawValue string
		expected bool
	}{
		{name: "true literal", rawValue: "true", expected: true},
		{name: "false literal", rawValue: "false", expected: false},
		{name: "yes literal", rawValue: "yes", expected: true},
		{name: "no literal", rawValue: "no", expected: false},
		{name: "on literal", rawValue: "on", expected: true},
		{name: "off literal", rawValue: "off", expected: false},
		{name: "one literal", rawValue: "1", expected: true},
		{name: "zero literal", rawValue: "0", expected: false},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("TEST_GET_BOOL", testCase.rawValue)

			actual := GetBool("TEST_GET_BOOL", false)
			if actual != testCase.expected {
				t.Fatalf("GetBool(%q) = %t, expected %t", testCase.rawValue, actual, testCase.expected)
			}
		})
	}
}

func TestGetBoolUnsupportedValuesFallback(t *testing.T) {
	testCases := []struct {
		name     string
		rawValue string
		fallback bool
	}{
		{name: "t literal", rawValue: "t", fallback: false},
		{name: "f literal", rawValue: "f", fallback: true},
		{name: "invalid literal", rawValue: "invalid", fallback: false},
		{name: "empty literal", rawValue: "", fallback: true},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("TEST_GET_BOOL", testCase.rawValue)

			actual := GetBool("TEST_GET_BOOL", testCase.fallback)
			if actual != testCase.fallback {
				t.Fatalf("GetBool(%q, fallback=%t) = %t, expected fallback", testCase.rawValue, testCase.fallback, actual)
			}
		})
	}
}
