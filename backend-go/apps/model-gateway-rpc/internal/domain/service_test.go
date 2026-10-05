package domain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher("0123456789abcdef-model-key")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := c.Encrypt("sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "sk-secret" {
		t.Fatal("key was not encrypted")
	}
	plain, err := c.Decrypt(encrypted)
	if err != nil || plain != "sk-secret" {
		t.Fatalf("plain=%q err=%v", plain, err)
	}
}
func TestCipherRejectsShortKey(t *testing.T) {
	if _, err := NewCipher("short"); err == nil {
		t.Fatal("expected short key rejection")
	}
}

func TestValidateBaseURLBlocksInternalTargets(t *testing.T) {
	for _, target := range []string{"http://127.0.0.1:8000", "http://169.254.169.254/latest", "http://localhost/v1", "http://metadata.google.internal"} {
		if err := validateBaseURL(target); err == nil {
			t.Fatalf("expected %s to be blocked", target)
		}
	}
	if err := validateBaseURL("https://api.deepseek.com/v1"); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAICompatibleCallSendsBearerAndParsesUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "test-model" {
			t.Fatalf("unexpected model: %v", body["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"ok\"}"}}],"usage":{"prompt_tokens":12,"completion_tokens":8}}`))
	}))
	defer server.Close()

	cipher, err := NewCipher("0123456789abcdef-model-key")
	if err != nil {
		t.Fatal(err)
	}
	key, err := cipher.Encrypt("sk-test")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(nil, cipher)
	service.http = server.Client()
	content, promptTokens, completionTokens, err := service.call(context.Background(), configRow{BaseURL: server.URL + "/v1", Model: "test-model", EncryptedKey: key, Timeout: 2, MaxTokens: 128}, "system", "prompt", "json_object")
	if err != nil {
		t.Fatal(err)
	}
	if content != `{"summary":"ok"}` || promptTokens != 12 || completionTokens != 8 {
		t.Fatalf("unexpected response: %q %d %d", content, promptTokens, completionTokens)
	}
}
