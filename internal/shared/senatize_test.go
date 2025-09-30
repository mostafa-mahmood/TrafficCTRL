package shared

import (
	"strings"
	"testing"
)

func TestSanitizeRedisKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "T01_ValidInput",
			input:    "user_123-api.tenant:us@corp",
			expected: "user_123-api.tenant:us@corp",
		},
		{
			name:     "T02_InjectionAttempt_ControlChars",
			input:    "key\x00\x01\n\r\t\x7fwith\x09control",
			expected: "keywithcontrol",
		},
		{
			name:     "T03_InjectionAttempt_RedisSyntax",
			input:    "user-123*FLUSHALL$\"newkey\"",
			expected: "user-123FLUSHALLnewkey",
		},
		{
			name:     "T04_InjectionAttempt_Whitespace",
			input:    "user 456 \t key\n",
			expected: "user456key",
		},
		{
			name:     "T05_MixedAllowedAndDisallowed",
			input:    "tenant_A.1.B:X-id/value?&",
			expected: "tenant_A.1.B:X-idvalue",
		},
		{
			name:     "T06_EmptyString",
			input:    "",
			expected: "",
		},
		{
			name:     "T07_OnlyDisallowedChars",
			input:    " \t\n!@#$%^&*()+=[]{}|\\;:'\",<>/?`~",
			expected: "@:",
		},
		{
			name:     "T08_LengthLimit",
			input:    strings.Repeat("a", 130),
			expected: strings.Repeat("a", 128),
		},
		{
			name:     "T09_ValidNonAscii_Letters",
			input:    "MöstaFâ",
			expected: "MöstaFâ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := sanitizeRedisKey(tt.input)
			if actual != tt.expected {
				t.Errorf("sanitizeRedisKey(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}
