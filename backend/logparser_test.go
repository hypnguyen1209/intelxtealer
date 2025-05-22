package main

import (
	"reflect"
	"testing"
)

func TestSplitLogLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Standard URL with login path",
			input:    "https://auralia.cloud/login:Bengalar:Robert2024!",
			expected: []string{"https://auralia.cloud/login", "Bengalar", "Robert2024!"},
		},
		{
			name:     "URL with multiple path segments",
			input:    "https://example.com/login/auth:username:pass123",
			expected: []string{"https://example.com/login/auth", "username", "pass123"},
		},
		{
			name:     "Simple domain with no path",
			input:    "https://example.com:user123:password456",
			expected: []string{"https://example.com", "user123", "password456"},
		},
		{
			name:     "Password containing colons",
			input:    "https://site.com:username:pass:with:colons",
			expected: []string{"https://site.com", "username", "pass:with:colons"},
		},
		{
			name:     "HTTP URL (not HTTPS)",
			input:    "http://insecure.com:myusername:simplepass",
			expected: []string{"http://insecure.com", "myusername", "simplepass"},
		},
		{
			name:     "URL with port number",
			input:    "https://example.com:8080:portuser:portpass",
			expected: []string{"https://example.com:8080", "portuser", "portpass"},
		}, {
			name:     "URL with query parameters",
			input:    "https://search.com/path?query=test:searchuser:searchpass",
			expected: []string{"https://search.com/path?query=test", "searchuser", "searchpass"},
		},
		{
			name:     "Android scheme URL",
			input:    "android://gNDQRvwT2GhkTMztoIx0GgXEEXR6GCnBN3MAHPuOa5w7LcsCcxLQY-1lxuyQqKSLxWjn9GqImVc2M1yoASB7Eg==@com.bnb.paynearby/:9047161186:Jumaila@06",
			expected: []string{"android://gNDQRvwT2GhkTMztoIx0GgXEEXR6GCnBN3MAHPuOa5w7LcsCcxLQY-1lxuyQqKSLxWjn9GqImVc2M1yoASB7Eg==@com.bnb.paynearby/", "9047161186", "Jumaila@06"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLogLine(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitLogLine(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
