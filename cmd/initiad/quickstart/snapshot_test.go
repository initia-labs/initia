package quickstart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadAndExtractSnapshot_InvalidURL(t *testing.T) {
	err := downloadAndExtractSnapshot("not-a-url", "/tmp/fakehome")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid snapshot URL")
}

func TestDownloadAndExtractSnapshot_FTPScheme(t *testing.T) {
	err := downloadAndExtractSnapshot("ftp://example.com/file.tar.lz4", "/tmp/fakehome")
	require.Error(t, err)
	require.Contains(t, err.Error(), "http")
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple comma separated",
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    " a , b , c ",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "single value",
			input:    "single",
			expected: []string{"single"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "empty elements",
			input:    "a,,b",
			expected: []string{"a", "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := splitAndTrim(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
