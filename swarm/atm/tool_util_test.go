package atm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseMimeType tests the ParseMimeType function
func TestParseMimeType(t *testing.T) {
	tests := []struct {
		input            string
		expectedData     string
		expectedMimeType string
	}{
		{
			input:            "#! Content --mime-type text/plain\nThis is a simple text.",
			expectedData:     "This is a simple text.",
			expectedMimeType: "text/plain",
		},
		{
			input:            "Description: text file --mime_type=text/csv\nName,Age\nJohn,23",
			expectedData:     "Name,Age\nJohn,23",
			expectedMimeType: "text/csv",
		},
		{
			input:            "//Content --mime_type text/plain\nThis is a simple text 2.",
			expectedData:     "This is a simple text 2.",
			expectedMimeType: "text/plain",
		},
		{
			input:            "Description: text file --mime-type=text/csv\nName,Age\nJohn,24",
			expectedData:     "Name,Age\nJohn,24",
			expectedMimeType: "text/csv",
		},
		{
			input:            "Content mime_type=text/plain\nThis is a simple text 3.",
			expectedData:     "This is a simple text 3.",
			expectedMimeType: "text/plain",
		},
		{
			input:            "Description: text file mime-type=text/csv\nName,Age\nJohn,25",
			expectedData:     "Name,Age\nJohn,25",
			expectedMimeType: "text/csv",
		},
		{
			input:            "No mime-type\nJust some text.",
			expectedData:     "No mime-type\nJust some text.",
			expectedMimeType: "",
		},
	}

	for _, tt := range tests {
		data, mimeType := ParseMimeType(tt.input)
		assert.Equal(t, tt.expectedData, data)
		assert.Equal(t, tt.expectedMimeType, mimeType, "Expected mime type does not match")
	}
}
