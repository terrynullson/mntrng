package worker

import "testing"

func TestValidateExternalURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		rawURL       string
		allowPrivate bool
		expectError  bool
	}{
		{
			name:        "rejects invalid scheme",
			rawURL:      "ftp://8.8.8.8/stream.m3u8",
			expectError: true,
		},
		{
			name:        "rejects loopback ip",
			rawURL:      "http://127.0.0.1/stream.m3u8",
			expectError: true,
		},
		{
			name:        "rejects private ip",
			rawURL:      "http://10.1.2.3/stream.m3u8",
			expectError: true,
		},
		{
			name:        "rejects localhost host",
			rawURL:      "http://localhost/stream.m3u8",
			expectError: true,
		},
		{
			name:        "allows public ip with https",
			rawURL:      "https://8.8.8.8/stream.m3u8",
			expectError: false,
		},
		{
			name:         "allow_private_toggle_allows_private_hosts",
			rawURL:       "http://localhost/stream.m3u8",
			allowPrivate: true,
			expectError:  false,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := validateExternalURL(testCase.rawURL, testCase.allowPrivate)
			if testCase.expectError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !testCase.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
