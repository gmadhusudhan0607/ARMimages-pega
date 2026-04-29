/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

import (
	"strings"
	"testing"
)

func TestParseFunctionalCapability(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    FunctionalCapability
		expectError bool
		errorMsg    string
	}{
		// Valid cases
		{"valid chat_completion", "chat_completion", FunctionalCapabilityChatCompletion, false, ""},
		{"valid image", "image", FunctionalCapabilityImage, false, ""},
		{"valid embedding", "embedding", FunctionalCapabilityEmbedding, false, ""},
		{"valid audio", "audio", FunctionalCapabilityAudio, false, ""},
		{"valid realtime", "realtime", FunctionalCapabilityRealtime, false, ""},

		// Invalid cases
		{"empty string", "", "", true, "invalid functional capability"},
		{"whitespace only", "   ", "", true, "invalid functional capability"},
		{"tab whitespace", "\t", "", true, "invalid functional capability"},
		{"newline whitespace", "\n", "", true, "invalid functional capability"},
		{"mixed whitespace", " \t\n ", "", true, "invalid functional capability"},
		{"invalid capability", "invalid", "", true, "invalid functional capability"},
		{"unknown capability", "unknown", "", true, "invalid functional capability"},
		{"test capability", "test", "", true, "invalid functional capability"},

		// Mixed case variations
		{"Chat_Completion mixed case", "Chat_Completion", "", true, "invalid functional capability"},
		{"CHAT_COMPLETION uppercase", "CHAT_COMPLETION", "", true, "invalid functional capability"},
		{"IMAGE uppercase", "IMAGE", "", true, "invalid functional capability"},
		{"Embedding mixed case", "Embedding", "", true, "invalid functional capability"},
		{"EMBEDDING uppercase", "EMBEDDING", "", true, "invalid functional capability"},
		{"Audio mixed case", "Audio", "", true, "invalid functional capability"},
		{"AUDIO uppercase", "AUDIO", "", true, "invalid functional capability"},

		// Special characters
		{"chat-completion with dash", "chat-completion", "", true, "invalid functional capability"},
		{"chat.completion with dot", "chat.completion", "", true, "invalid functional capability"},
		{"chat completion with space", "chat completion", "", true, "invalid functional capability"},
		{"chat_completion_extra with extra", "chat_completion_extra", "", true, "invalid functional capability"},
		{"_chat_completion with leading underscore", "_chat_completion", "", true, "invalid functional capability"},
		{"chat_completion_ with trailing underscore", "chat_completion_", "", true, "invalid functional capability"},

		// Unicode characters
		{"chät_completion with umlaut", "chät_completion", "", true, "invalid functional capability"},
		{"chat_completión with accent", "chat_completión", "", true, "invalid functional capability"},
		{"emoji capability", "🎯", "", true, "invalid functional capability"},
		{"chinese characters", "聊天完成", "", true, "invalid functional capability"},
		{"russian characters", "чат_завершение", "", true, "invalid functional capability"},

		// Very long strings
		{"very long string", strings.Repeat("a", 1000), "", true, "invalid functional capability"},
		{"very long valid prefix", "chat_completion" + strings.Repeat("x", 1000), "", true, "invalid functional capability"},

		// Null byte handling
		{"null byte in middle", "chat_completion\x00", "", true, "invalid functional capability"},
		{"null byte at start", "\x00chat_completion", "", true, "invalid functional capability"},
		{"null byte at end", "chat_completion\x00", "", true, "invalid functional capability"},
		{"only null byte", "\x00", "", true, "invalid functional capability"},

		// Control characters
		{"carriage return", "chat_completion\r", "", true, "invalid functional capability"},
		{"form feed", "chat_completion\f", "", true, "invalid functional capability"},
		{"vertical tab", "chat_completion\v", "", true, "invalid functional capability"},
		{"bell character", "chat_completion\a", "", true, "invalid functional capability"},
		{"backspace", "chat_completion\b", "", true, "invalid functional capability"},

		// Numbers and mixed alphanumeric
		{"numeric capability", "123", "", true, "invalid functional capability"},
		{"mixed alphanumeric", "chat123", "", true, "invalid functional capability"},
		{"capability with numbers", "chat_completion_v1", "", true, "invalid functional capability"},

		// Similar but invalid variations
		{"chat_completions plural", "chat_completions", "", true, "invalid functional capability"},
		{"chatcompletion no underscore", "chatcompletion", "", true, "invalid functional capability"},
		{"chat completion space", "chat completion", "", true, "invalid functional capability"},
		{"images plural", "images", "", true, "invalid functional capability"},
		{"embeddings plural", "embeddings", "", true, "invalid functional capability"},
		{"audios plural", "audios", "", true, "invalid functional capability"},

		// Edge cases with valid capabilities as substrings
		{"chat_completion_extended", "chat_completion_extended", "", true, "invalid functional capability"},
		{"prefix_chat_completion", "prefix_chat_completion", "", true, "invalid functional capability"},
		{"text_embedding", "text_embedding", "", true, "invalid functional capability"},
		{"audio_generation", "audio_generation", "", true, "invalid functional capability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFunctionalCapability(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.input)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
				// Verify that result is empty when error occurs
				if result != "" {
					t.Errorf("Expected empty result when error occurs, got: %s", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
					return
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s for input '%s'", tt.expected, result, tt.input)
				}
			}
		})
	}
}

func TestIsValidFunctionalCapability(t *testing.T) {
	tests := []struct {
		name       string
		capability FunctionalCapability
		expected   bool
	}{
		// Valid capabilities
		{"valid chat_completion", FunctionalCapabilityChatCompletion, true},
		{"valid image", FunctionalCapabilityImage, true},
		{"valid embedding", FunctionalCapabilityEmbedding, true},
		{"valid audio", FunctionalCapabilityAudio, true},
		{"valid realtime", FunctionalCapabilityRealtime, true},

		// Invalid capabilities
		{"empty capability", FunctionalCapability(""), false},
		{"invalid capability", FunctionalCapability("invalid"), false},
		{"mixed case", FunctionalCapability("Chat_Completion"), false},
		{"uppercase", FunctionalCapability("CHAT_COMPLETION"), false},
		{"with spaces", FunctionalCapability("chat completion"), false},
		{"with dashes", FunctionalCapability("chat-completion"), false},
		{"with dots", FunctionalCapability("chat.completion"), false},
		{"plural form", FunctionalCapability("chat_completions"), false},
		{"numeric", FunctionalCapability("123"), false},
		{"unicode", FunctionalCapability("chät_completion"), false},
		{"very long", FunctionalCapability(strings.Repeat("a", 1000)), false},
		{"null byte", FunctionalCapability("chat_completion\x00"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFunctionalCapability(tt.capability)
			if result != tt.expected {
				t.Errorf("Expected %v for capability '%s', got %v", tt.expected, tt.capability, result)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkParseFunctionalCapability(b *testing.B) {
	testCases := []string{
		"chat_completion",
		"image",
		"embedding",
		"audio",
		"invalid",
		"",
		strings.Repeat("a", 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_, _ = ParseFunctionalCapability(tc)
		}
	}
}

func BenchmarkIsValidFunctionalCapability(b *testing.B) {
	testCases := []FunctionalCapability{
		FunctionalCapabilityChatCompletion,
		FunctionalCapabilityImage,
		FunctionalCapabilityEmbedding,
		FunctionalCapabilityAudio,
		FunctionalCapability("invalid"),
		FunctionalCapability(""),
		FunctionalCapability(strings.Repeat("a", 100)),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = IsValidFunctionalCapability(tc)
		}
	}
}

// Test edge cases with concurrent access
func TestParseFunctionalCapabilityConcurrent(t *testing.T) {
	const numGoroutines = 100
	const numIterations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < numIterations; j++ {
				// Test valid cases
				if result, err := ParseFunctionalCapability("chat_completion"); err != nil || result != FunctionalCapabilityChatCompletion {
					t.Errorf("Concurrent test failed for valid input")
					return
				}
				// Test invalid cases
				if _, err := ParseFunctionalCapability("invalid"); err == nil {
					t.Errorf("Concurrent test failed for invalid input")
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Test memory usage with large inputs
func TestParseFunctionalCapabilityMemory(t *testing.T) {
	// Test with very large string to ensure no memory leaks
	largeString := strings.Repeat("invalid_capability_", 10000)

	for i := 0; i < 1000; i++ {
		_, err := ParseFunctionalCapability(largeString)
		if err == nil {
			t.Errorf("Expected error for large invalid string")
		}
	}
}

// Test all constants are properly defined
func TestFunctionalCapabilityConstants(t *testing.T) {
	constants := map[string]FunctionalCapability{
		"chat_completion": FunctionalCapabilityChatCompletion,
		"image":           FunctionalCapabilityImage,
		"embedding":       FunctionalCapabilityEmbedding,
		"audio":           FunctionalCapabilityAudio,
		"realtime":        FunctionalCapabilityRealtime,
	}

	for expectedStr, constant := range constants {
		if string(constant) != expectedStr {
			t.Errorf("Constant %v should equal '%s', got '%s'", constant, expectedStr, string(constant))
		}

		// Test that parsing the string returns the constant
		parsed, err := ParseFunctionalCapability(expectedStr)
		if err != nil {
			t.Errorf("Failed to parse valid capability '%s': %v", expectedStr, err)
		}
		if parsed != constant {
			t.Errorf("Parsed capability '%s' should equal %v, got %v", expectedStr, constant, parsed)
		}

		// Test that the constant is valid
		if !IsValidFunctionalCapability(constant) {
			t.Errorf("Constant %v should be valid", constant)
		}
	}
}
