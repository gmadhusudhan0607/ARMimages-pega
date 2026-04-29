/*
* Copyright (c) 2025 Pegasystems Inc.
* All rights reserved.
 */

package load

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"
)

//go:embed buddy_prompts/haiku3_buddy.json
var haiku3_buddy_prompt string

//go:embed buddy_prompts/nova-lite_buddy.json
var nova_lite_buddy_prompt string

// list of  tokens to randomize
var tokens = []string{
	"add_your_sax_token_here",
} //TODO: integrate SAX library, so we don't have to add tokens manually

// hostUrl for all requests
var hostUrl = "http://localhost:8080"

type modelCall struct {
	url    string
	body   string
	labels []string
}

var gatewayVersion = "v1.45"

// list of target URLs to randomize
var models = []modelCall{
	{
		hostUrl + "/openai/deployments/gpt-4o-mini/chat/completions?api-version=2024-02-01",
		`{ "messages": [ { "role": "user", "content":  "Can you tell me the history of the world, including what are the most recent findings about the Big Bang and then about terraformation, the separation of contients, appearance of life, the humans and today's threats to the world? Take your time, count until %d, write in chapters that are about 500 words long." }], "stream": true }`,
		[]string{"openai", "stream", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-4o-mini/chat/completions?api-version=2024-02-01",
		`{ "messages": [ { "role": "user", "content":  "Can you tell me the history of the world, including what are the most recent findings about the Big Bang and then about terraformation, the separation of contients, appearance of life, the humans and today's threats to the world? Take your time, count until %d, write in chapters that are about 500 words long." }] }`,
		[]string{"openai", "chat", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01",
		`{ "messages": [ {"role": "user", "content": "Tell me a joke about %d cats." } ]}`,
		[]string{"openai", "chat", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-5.1/chat/completions?api-version=2024-02-01",
		`{ "messages": [ {"role": "user", "content": "Explain quantum computing in %d words" }]}`,
		[]string{"openai", "chat", "gpt5", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-5.1/chat/completions?api-version=2024-02-01",
		`{ "messages": [ {"role": "user", "content": "Explain quantum computing in %d words" }], "stream": true}`,
		[]string{"openai", "stream", "gpt5", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-5.2/chat/completions?api-version=2024-02-01",
		`{ "messages": [ {"role": "user", "content": "Write a haiku about %d clouds" }]}`,
		[]string{"openai", "chat", "gpt5", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/gpt-5.2/chat/completions?api-version=2024-02-01",
		`{ "messages": [ {"role": "user", "content": "Write a haiku about %d clouds" }], "stream": true}`,
		[]string{"openai", "stream", "gpt5", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/text-embedding-ada-002/embeddings?api-version=2024-02-01",
		`{"input":"Analyze this text %d."}`,
		[]string{"openai", "embedding", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-haiku/chat/completions",
		haiku3_buddy_prompt,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-haiku/converse",
		haiku3_buddy_prompt,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-haiku/converse-stream",
		haiku3_buddy_prompt,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-5-sonnet/chat/completions",
		`{"modelId": "anthropic.claude-3-5-sonnet-20241022-v2:0", "messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-5-sonnet/converse",
		`{"modelId": "anthropic.claude-3-5-sonnet-20241022-v2:0", "messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-7-sonnet/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-3-7-sonnet/converse-stream",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-sonnet-4-5/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-sonnet-4-5/converse-stream",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-haiku-4-5/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-haiku-4-5/converse-stream",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-sonnet-4-6/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-sonnet-4-6/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-opus-4-6/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/anthropic/deployments/claude-opus-4-6/converse",
		`{"messages": [{"role": "user", "content": [{"text": "Tell me a joke about year %d"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/meta/deployments/llama3-8b-instruct/chat/completions",
		`{"modelId": "meta.llama3-8b-instruct-v1:0", "messages": [{"role": "user", "content": [{"text": "Tell me a joke about %d cats"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/meta/deployments/llama3-8b-instruct/converse",
		`{"modelId": "meta.llama3-8b-instruct-v1:0", "messages": [{"role": "user", "content": [{"text": "Tell me a joke about %d cats"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/meta/deployments/llama3-8b-instruct/converse-stream",
		`{"modelId": "meta.llama3-8b-instruct-v1:0", "messages": [{"role": "user", "content": [{"text": "Tell me a joke about %d cats"}]}]}`,
		[]string{"bedrock", "chat", gatewayVersion},
	},
	{
		hostUrl + "/amazon/deployments/nova-lite-v1/converse",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-lite-v1/converse-stream",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-micro-v1/converse",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-micro-v1/converse-stream",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-pro-v1/converse",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-pro-v1/converse-stream",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-lite-v1/converse",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-lite-v1/converse-stream",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-pro-preview/converse",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-pro-preview/converse-stream",
		nova_lite_buddy_prompt,
		[]string{"bedrock", "chat"},
	},
	{
		hostUrl + "/amazon/deployments/titan-embed-text/embeddings",
		`{"inputText":"Analyze this text %d."}`,
		[]string{"bedrock", "embedding", "titan", gatewayVersion},
	},
	{
		hostUrl + "/amazon/deployments/titan-embed-text/invoke",
		`{"inputText":"Analyze this text %d."}`,
		[]string{"bedrock", "embedding", "titan", gatewayVersion},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-multimodal-embeddings/embeddings",
		`{"taskType": "SINGLE_EMBEDDING", "singleEmbeddingParams": {"embeddingPurpose": "GENERIC_INDEX","embeddingDimension": 3072,"text": {"truncationMode": "END", "value": "Hello, World!"}}}`,
		[]string{"bedrock", "embedding", "titan", gatewayVersion},
	},
	{
		hostUrl + "/amazon/deployments/nova-2-multimodal-embeddings/invoke",
		`{"taskType": "SINGLE_EMBEDDING", "singleEmbeddingParams": {"embeddingPurpose": "GENERIC_INDEX","embeddingDimension": 3072,"text": {"truncationMode": "END", "value": "Hello, World!"}}}`,
		[]string{"bedrock", "embedding", "titan", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-2.0-flash/chat/completions",
		`{ "model": "google/gemini-2.0-flash", "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-2.0-flash/chat/completions",
		`{ "model": "google/gemini-2.0-flash", "stream": true, "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-2.5-flash/chat/completions",
		`{ "model": "google/gemini-2.5-flash", "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-2.5-flash/chat/completions",
		`{ "model": "google/gemini-2.5-flash", "stream": true, "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-3.0-pro-preview/chat/completions",
		`{ "model": "google/gemini-3.0-pro-preview", "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-3.0-pro-preview/chat/completions",
		`{ "model": "google/gemini-3.0-pro-preview", "stream": true, "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-3.0-flash-preview/chat/completions",
		`{ "model": "google/gemini-3.0-flash-preview", "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-3.0-flash-preview/chat/completions",
		`{ "model": "google/gemini-3.0-flash-preview", "stream": true, "messages": [ {"role": "user","content": "Write me a poem with %d words?"}]}`,
		[]string{"gemini", "chat", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/text-multilingual-embedding-002/embeddings?api-version=2024-02-01",
		`{ "model": "text-multilingual-embedding-002", "texts":["Analyze this text %d."] }`,
		[]string{"gemini", "embedding", gatewayVersion},
	},
	{
		hostUrl + "/openai/deployments/dall-e-3/images/generations?api-version=2024-02-01",
		`{ "prompt": "Draw the number %d",  "n": 1,  "size": "1024x1024",  "response_format": "url",  "user": "user123456",  "quality": "standard",  "style": "vivid" }`,
		[]string{"openai", "image", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/imagen-3/images/generations?api-version=2024-02-01",
		`{ "modelId": "imagen-3.0-generate-002", "payload": { "prompt": "Draw the number %d.", "number_of_images": 1, "aspect_ratio": "1:1",    "safety_filter_level": "block_some",    "person_generation": "allow_all"  }}`,
		[]string{"google", "image", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/imagen-4.0/images/generations?api-version=2024-02-01",
		`{ "modelId": "imagen-4.0-generate-001", "payload": { "prompt": "Draw the number %d.", "number_of_images": 1, "aspect_ratio": "1:1",    "safety_filter_level": "block_some",    "person_generation": "allow_all"  }}`,
		[]string{"google", "image", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/imagen-4.0-fast/images/generations?api-version=2024-02-01",
		`{ "modelId": "imagen-4.0-fast-generate-001", "payload": { "prompt": "Draw the number %d.", "number_of_images": 1, "aspect_ratio": "1:1",    "safety_filter_level": "block_some",    "person_generation": "allow_all"  }}`,
		[]string{"google", "image", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/imagen-4.0-ultra/images/generations?api-version=2024-02-01",
		`{ "modelId": "imagen-4.0-ultra-generate-001", "payload": { "prompt": "Draw the number %d.", "number_of_images": 1, "aspect_ratio": "1:1",    "safety_filter_level": "block_some",    "person_generation": "allow_all"  }}`,
		[]string{"google", "image", gatewayVersion},
	},
	{
		hostUrl + "/google/deployments/gemini-3.1-flash-image-preview/generateContent",
		`{"contents":[{"role":"user","parts":[{"text":"A beautiful sunset over mountains with %d vibrant colors"}]}],"generationConfig":{"responseModalities":["IMAGE"]}}`,
		[]string{"gemini", "image", "generateContent", gatewayVersion},
	},
}

func TestRandomizedPostRequests(t *testing.T) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	var wg sync.WaitGroup
	numCalls := 5000

	// filter models by label, like "bedrock" or "openai" - or not
	filter := "bedrock"
	var filteredModels []modelCall
	if len(filter) > 0 {
		for _, m := range models {
			if slices.Contains(m.labels, filter) {
				filteredModels = append(filteredModels, m)
			}
		}
	} else {
		filteredModels = models
	}

	for i := 0; i < numCalls; i++ {
		//delay := time.Duration(rand.Intn(2000)+500) * time.Millisecond
		delay := 10000 * time.Millisecond
		time.Sleep(delay)
		wg.Add(1)
		go func() {
			defer wg.Done()

			// select random URL and token
			m := filteredModels[rand.Intn(len(filteredModels))]
			if call(t, m) {
				return
			}
		}()
	}

	wg.Wait()
}

func TestSequentialPostRequests(t *testing.T) {

	// filter models by label, like "bedrock" or "openai" - or empty string ""
	filter := "bedrock"
	var filteredModels []modelCall
	for _, m := range models {
		for _, label := range m.labels {
			if label == filter || filter == "" {
				filteredModels = append(filteredModels, m)
				break
			}
		}
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	roundsOfCalls := 50 // each round calls all models that match the filter once.
	for i := 0; i < roundsOfCalls; i++ {
		for _, m := range filteredModels {
			if call(t, m) {
				return
			}
		}
	}
}

func call(t *testing.T, m modelCall) bool {
	// select random token to create metrics for different isolations in grafana
	token := tokens[rand.Intn(len(tokens))]
	b := fmt.Sprintf(m.body, rand.Intn(100))
	req, err := http.NewRequest("POST", m.url, bytes.NewBuffer([]byte(b)))
	if err != nil {
		t.Errorf("Error creating request: %v", err)
		return true
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	now := time.Now()
	resp, err := client.Do(req)
	rt := time.Since(now)
	if err != nil {
		t.Errorf("Error sending request: %v", err)
		return true
	}
	defer resp.Body.Close()

	//tee := io.TeeReader(resp.Body, os.Stdout)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response body: %v", err)
		return true
	}

	// remove line breaks and carriage returns
	oneline := bytes.ReplaceAll(body, []byte("\n"), []byte(""))
	oneline = bytes.ReplaceAll(oneline, []byte("\r"), []byte(""))

	// Remove control characters
	re := regexp.MustCompile(`[^a-zA-Z0-9\p{P}\s]+`)
	oneline = []byte(re.ReplaceAllString(string(oneline), ""))

	// Truncate response sample
	trunc := oneline
	if len(trunc) > 100 {
		trunc = trunc[:200]
	}

	cl := resp.Header.Get("Content-Length")
	resp_time_sec := rt.Seconds()

	s := fmt.Sprintf("%d; %f; %s; %s; %s", resp.StatusCode, resp_time_sec, m.url, cl, string(trunc))
	t.Log(s)
	return false
}
