package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	localAPIURL  string
	localModel   string
	localStream  bool
	localTimeout int
	localBackend string // "auto", "node-llm", "ollama"
)

// localCmd represents the local AI command group
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Local AI model operations (node-llm, OpenAI-compatible)",
	Long: `Commands for testing and using locally running AI models.

Supports:
- node-llm (OpenAI-compatible API) - PRIMARY
- Any OpenAI-compatible local endpoint
- Ollama (legacy, fallback)

The armyknife-code fork uses node-llm which provides an OpenAI-compatible API.

Examples:
  armyknife local status
  armyknife local models
  armyknife local chat "Explain this code" --model gpt-4
  armyknife local generate "Write a function to sort an array"
  armyknife local test --model phi3`,
}

// localStatusCmd checks local AI status
var localStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check local AI service status",
	Long:  `Check if the local AI service (node-llm) is running and accessible.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🔍 Checking local AI status...\n")
		fmt.Printf("   URL: %s\n", localAPIURL)
		fmt.Printf("   Backend: %s\n\n", localBackend)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}

		// Try OpenAI-compatible endpoint first (node-llm)
		if localBackend == "auto" || localBackend == "node-llm" {
			resp, err := client.Get(localAPIURL + "/v1/models")
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var models map[string]interface{}
					if json.NewDecoder(resp.Body).Decode(&models) == nil {
						fmt.Printf("✅ node-llm (OpenAI-compatible) is running!\n\n")
						if data, ok := models["data"].([]interface{}); ok {
							fmt.Printf("📦 Available Models (%d):\n", len(data))
							for _, m := range data {
								if model, ok := m.(map[string]interface{}); ok {
									fmt.Printf("   - %s\n", model["id"])
								}
							}
						}
						return
					}
				}
			}
		}

		// Fallback to Ollama check
		if localBackend == "auto" || localBackend == "ollama" {
			ollamaURL := strings.Replace(localAPIURL, "/v1", "", 1)
			if !strings.Contains(ollamaURL, ":11434") {
				ollamaURL = "http://localhost:11434"
			}
			resp, err := client.Get(ollamaURL + "/api/tags")
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var result map[string]interface{}
					if json.NewDecoder(resp.Body).Decode(&result) == nil {
						fmt.Printf("✅ Ollama is running!\n\n")
						if models, ok := result["models"].([]interface{}); ok {
							fmt.Printf("📦 Installed Models (%d):\n", len(models))
							for _, m := range models {
								if model, ok := m.(map[string]interface{}); ok {
									name := model["name"].(string)
									size := ""
									if s, ok := model["size"].(float64); ok {
										size = fmt.Sprintf(" (%.1f GB)", s/1024/1024/1024)
									}
									fmt.Printf("   - %s%s\n", name, size)
								}
							}
						}
						return
					}
				}
			}
		}

		fmt.Printf("❌ Cannot connect to local AI service\n")
		fmt.Printf("   Tried: %s\n\n", localAPIURL)
		fmt.Println("Make sure node-llm or the AI backend is running:")
		fmt.Println("  1. Start armyknife-code (VS Code fork)")
		fmt.Println("  2. Check that the AI service is enabled")
		fmt.Println("  3. Verify the URL with --api-url flag")
	},
}

// localModelsCmd lists available models
var localModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available local models",
	Long:  `List all models available in the local AI service.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("📦 Local Models (%s)\n", localAPIURL)
		fmt.Println(strings.Repeat("-", 50))

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}

		// Try OpenAI-compatible endpoint (node-llm)
		resp, err := client.Get(localAPIURL + "/v1/models")
		if err == nil {
			defer resp.Body.Close()
			var result map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				if data, ok := result["data"].([]interface{}); ok {
					if len(data) == 0 {
						fmt.Println("No models available.")
						return
					}
					for _, m := range data {
						if model, ok := m.(map[string]interface{}); ok {
							id := model["id"].(string)
							owned := ""
							if owner, ok := model["owned_by"].(string); ok {
								owned = fmt.Sprintf(" (by %s)", owner)
							}
							fmt.Printf("  %s%s\n", id, owned)
						}
					}
					return
				}
			}
		}

		// Fallback to Ollama
		ollamaURL := "http://localhost:11434"
		resp, err = client.Get(ollamaURL + "/api/tags")
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("❌ Error parsing response: %v\n", err)
			return
		}

		if models, ok := result["models"].([]interface{}); ok {
			if len(models) == 0 {
				fmt.Println("No models installed.")
				return
			}
			for _, m := range models {
				model := m.(map[string]interface{})
				name := model["name"].(string)
				sizeStr := ""
				if size, ok := model["size"].(float64); ok {
					sizeGB := size / 1024 / 1024 / 1024
					sizeStr = fmt.Sprintf("%.1f GB", sizeGB)
				}
				fmt.Printf("  %-30s %10s\n", name, sizeStr)
			}
		}
	},
}

// localChatCmd sends a chat message using OpenAI-compatible API
var localChatCmd = &cobra.Command{
	Use:   "chat <message>",
	Short: "Chat with local AI model",
	Long: `Send a chat message to the local AI model using OpenAI-compatible API.

Examples:
  armyknife local chat "Explain this Go code"
  armyknife local chat "How do I implement a binary tree?" --model gpt-4
  armyknife local chat "Review this function for bugs" --stream`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message := args[0]

		fmt.Printf("💬 Chat with %s\n", localModel)
		fmt.Println(strings.Repeat("-", 50))

		// OpenAI-compatible request format
		reqBody := map[string]interface{}{
			"model": localModel,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"stream": localStream,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			localAPIURL+"/v1/chat/completions",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if localStream {
			// Handle SSE streaming response
			reader := resp.Body
			buf := make([]byte, 4096)
			for {
				n, err := reader.Read(buf)
				if err != nil {
					break
				}
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if data == "[DONE]" {
							break
						}
						var chunk map[string]interface{}
						if json.Unmarshal([]byte(data), &chunk) == nil {
							if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
								if choice, ok := choices[0].(map[string]interface{}); ok {
									if delta, ok := choice["delta"].(map[string]interface{}); ok {
										if content, ok := delta["content"].(string); ok {
											fmt.Print(content)
										}
									}
								}
							}
						}
					}
				}
			}
			fmt.Println()
		} else {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("❌ Error parsing response: %v\n", err)
				return
			}

			if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							fmt.Println(content)
						}
					}
				}
			}

			// Show usage info
			if usage, ok := result["usage"].(map[string]interface{}); ok {
				fmt.Printf("\n📊 Tokens: %v prompt, %v completion, %v total\n",
					usage["prompt_tokens"], usage["completion_tokens"], usage["total_tokens"])
			}
		}
	},
}

// localGenerateCmd generates text using OpenAI-compatible completions
var localGenerateCmd = &cobra.Command{
	Use:   "generate <prompt>",
	Short: "Generate text with local AI",
	Long: `Generate text completion from a prompt using OpenAI-compatible API.

Examples:
  armyknife local generate "// Function to calculate fibonacci"
  armyknife local generate "func sortSlice(s []int) []int {"
  armyknife local generate "Write unit tests for:" --model gpt-4`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompt := args[0]

		fmt.Printf("🤖 Generating with %s...\n\n", localModel)

		// Use chat completions endpoint (more widely supported)
		reqBody := map[string]interface{}{
			"model": localModel,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream": localStream,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			localAPIURL+"/v1/chat/completions",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if localStream {
			reader := resp.Body
			buf := make([]byte, 4096)
			for {
				n, err := reader.Read(buf)
				if err != nil {
					break
				}
				lines := strings.Split(string(buf[:n]), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if data == "[DONE]" {
							break
						}
						var chunk map[string]interface{}
						if json.Unmarshal([]byte(data), &chunk) == nil {
							if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
								if choice, ok := choices[0].(map[string]interface{}); ok {
									if delta, ok := choice["delta"].(map[string]interface{}); ok {
										if content, ok := delta["content"].(string); ok {
											fmt.Print(content)
										}
									}
								}
							}
						}
					}
				}
			}
			fmt.Println()
		} else {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				return
			}

			if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							fmt.Println(content)
						}
					}
				}
			}

			if usage, ok := result["usage"].(map[string]interface{}); ok {
				fmt.Printf("\n📊 Tokens: %v total\n", usage["total_tokens"])
			}
		}
	},
}

// localTestCmd runs a benchmark test
var localTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test local model with code tasks",
	Long: `Run a quick benchmark test on the local model with code-related tasks.

Tests:
1. Code completion
2. Code explanation
3. Bug detection`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🧪 Testing local model: %s\n", localModel)
		fmt.Printf("   URL: %s\n", localAPIURL)
		fmt.Println(strings.Repeat("=", 60))

		tests := []struct {
			name   string
			prompt string
		}{
			{
				name:   "Code Completion",
				prompt: "Complete this Go function:\n\nfunc fibonacci(n int) int {\n    // Return the nth fibonacci number",
			},
			{
				name:   "Code Explanation",
				prompt: "Explain what this code does in one sentence:\n\nfunc (s *Stack) Pop() interface{} {\n    if len(s.items) == 0 {\n        return nil\n    }\n    item := s.items[len(s.items)-1]\n    s.items = s.items[:len(s.items)-1]\n    return item\n}",
			},
			{
				name:   "Bug Detection",
				prompt: "Find the bug in this code:\n\nfunc divide(a, b int) int {\n    return a / b\n}",
			},
		}

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		totalTime := 0.0
		totalTokens := 0.0

		for i, test := range tests {
			fmt.Printf("\n%d. %s\n", i+1, test.name)
			fmt.Println(strings.Repeat("-", 40))

			reqBody := map[string]interface{}{
				"model": localModel,
				"messages": []map[string]string{
					{"role": "user", "content": test.prompt},
				},
				"max_tokens": 150,
			}

			jsonData, _ := json.Marshal(reqBody)
			start := time.Now()

			resp, err := client.Post(
				localAPIURL+"/v1/chat/completions",
				"application/json",
				bytes.NewBuffer(jsonData),
			)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				continue
			}

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			elapsed := time.Since(start)
			totalTime += elapsed.Seconds()

			if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							// Truncate long responses
							if len(content) > 300 {
								content = content[:300] + "..."
							}
							fmt.Println(content)
						}
					}
				}
			}

			if usage, ok := result["usage"].(map[string]interface{}); ok {
				if tokens, ok := usage["total_tokens"].(float64); ok {
					totalTokens += tokens
				}
			}

			fmt.Printf("⏱️  %.2fs\n", elapsed.Seconds())
		}

		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("\n📊 Summary\n")
		fmt.Printf("   Total Time: %.2fs\n", totalTime)
		fmt.Printf("   Total Tokens: %.0f\n", totalTokens)
		if totalTime > 0 {
			fmt.Printf("   Avg Speed: %.1f tokens/sec\n", totalTokens/totalTime)
		}
	},
}

// localEmbedCmd generates embeddings using OpenAI-compatible API
var localEmbedCmd = &cobra.Command{
	Use:   "embed <text>",
	Short: "Generate embeddings with local model",
	Long: `Generate vector embeddings for text using the local AI service.

Examples:
  armyknife local embed "function to sort array"
  armyknife local embed "authentication middleware" --model text-embedding-3-small`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		text := args[0]

		embeddingModel := localModel
		if !strings.Contains(localModel, "embed") {
			embeddingModel = "text-embedding-3-small" // Default embedding model
		}

		fmt.Printf("🧮 Generating embedding with %s\n", embeddingModel)

		reqBody := map[string]interface{}{
			"model": embeddingModel,
			"input": text,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			localAPIURL+"/v1/embeddings",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
			if item, ok := data[0].(map[string]interface{}); ok {
				if embedding, ok := item["embedding"].([]interface{}); ok {
					fmt.Printf("✅ Generated embedding\n")
					fmt.Printf("   Dimensions: %d\n", len(embedding))
					if len(embedding) >= 3 {
						fmt.Printf("   Preview: [%.4f, %.4f, %.4f, ...]\n",
							embedding[0], embedding[1], embedding[2])
					}
					return
				}
			}
		}

		fmt.Printf("❌ No embedding in response\n")
		body, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(body))
	},
}

// localHealthCmd checks all AI endpoints
var localHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of all local AI endpoints",
	Long: `Check the health status of all AI-related endpoints on the local service.

Tests:
- /v1/models - Model listing
- /v1/chat/completions - Chat endpoint
- /v1/embeddings - Embedding endpoint`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🏥 Health Check: %s\n", localAPIURL)
		fmt.Println(strings.Repeat("=", 60))

		client := &http.Client{Timeout: 10 * time.Second}

		endpoints := []struct {
			name   string
			method string
			path   string
			body   interface{}
		}{
			{"Models", "GET", "/v1/models", nil},
			{"Chat", "POST", "/v1/chat/completions", map[string]interface{}{
				"model":      localModel,
				"messages":   []map[string]string{{"role": "user", "content": "test"}},
				"max_tokens": 5,
			}},
			{"Embeddings", "POST", "/v1/embeddings", map[string]interface{}{
				"model": "text-embedding-3-small",
				"input": "test",
			}},
		}

		for _, ep := range endpoints {
			fmt.Printf("\n%-12s %s%s\n", ep.name+":", localAPIURL, ep.path)

			var resp *http.Response
			var err error

			if ep.method == "GET" {
				resp, err = client.Get(localAPIURL + ep.path)
			} else {
				jsonData, _ := json.Marshal(ep.body)
				resp, err = client.Post(localAPIURL+ep.path, "application/json", bytes.NewBuffer(jsonData))
			}

			if err != nil {
				fmt.Printf("   ❌ Error: %v\n", err)
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				fmt.Printf("   ✅ Status: %d\n", resp.StatusCode)
				// Parse and show brief response info
				var result map[string]interface{}
				if json.Unmarshal(body, &result) == nil {
					if data, ok := result["data"].([]interface{}); ok {
						fmt.Printf("   📊 Items: %d\n", len(data))
					}
					if usage, ok := result["usage"].(map[string]interface{}); ok {
						fmt.Printf("   📊 Tokens: %v\n", usage["total_tokens"])
					}
				}
			} else {
				fmt.Printf("   ❌ Status: %d\n", resp.StatusCode)
				fmt.Printf("   Response: %s\n", string(body)[:min(100, len(body))])
			}
		}

		fmt.Println(strings.Repeat("=", 60))
	},
}

// aiRouterCmd tests the AI router endpoint
var aiRouterCmd = &cobra.Command{
	Use:   "router <prompt>",
	Short: "Test AI router endpoint",
	Long: `Test the AI router endpoint that handles multi-model routing.

The router can send requests to:
- Local models (node-llm)
- Cloud providers (OpenAI, Anthropic)

Examples:
  armyknife local router "Explain this code" --model local
  armyknife local router "Complex analysis" --model cloud`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompt := args[0]
		routerURL := os.Getenv("AI_ROUTER_URL")
		if routerURL == "" {
			routerURL = "http://localhost:8080"
		}

		fmt.Printf("🔀 AI Router: %s\n", routerURL)
		fmt.Println(strings.Repeat("-", 50))

		reqBody := map[string]interface{}{
			"prompt": prompt,
			"model":  localModel,
			"context": map[string]string{
				"language": "go",
			},
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			routerURL+"/api/v1/ai/route",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("❌ Error parsing response: %v\n", err)
			return
		}

		if result["success"] == true {
			if data, ok := result["data"].(map[string]interface{}); ok {
				fmt.Printf("✅ Response from %s (%s):\n\n",
					data["provider"], data["model_used"])
				fmt.Println(data["response"])
				if latency, ok := data["latency_ms"].(float64); ok {
					fmt.Printf("\n⏱️  Latency: %.0fms\n", latency)
				}
			}
		} else {
			fmt.Printf("❌ Router error: %v\n", result["error"])
		}
	},
}

func init() {
	rootCmd.AddCommand(localCmd)

	// Local subcommands
	localCmd.AddCommand(localStatusCmd)
	localCmd.AddCommand(localModelsCmd)
	localCmd.AddCommand(localChatCmd)
	localCmd.AddCommand(localGenerateCmd)
	localCmd.AddCommand(localTestCmd)
	localCmd.AddCommand(localEmbedCmd)
	localCmd.AddCommand(localHealthCmd)
	localCmd.AddCommand(aiRouterCmd)

	// Global flags for local commands
	localCmd.PersistentFlags().StringVar(&localAPIURL, "api-url", "http://localhost:11434", "Local AI API URL (OpenAI-compatible)")
	localCmd.PersistentFlags().StringVar(&localModel, "model", "gpt-4", "Model to use")
	localCmd.PersistentFlags().BoolVar(&localStream, "stream", false, "Stream responses")
	localCmd.PersistentFlags().IntVar(&localTimeout, "timeout", 120, "Request timeout in seconds")
	localCmd.PersistentFlags().StringVar(&localBackend, "backend", "auto", "Backend type: auto, node-llm, ollama")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
