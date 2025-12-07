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
	localOllamaURL string
	localModel     string
	localStream    bool
	localTimeout   int
)

// localCmd represents the local AI command group
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Local AI model operations (Ollama, node-llm)",
	Long: `Commands for testing and using locally running AI models.

Supports:
- Ollama (default port 11434)
- node-llm (OpenAI-compatible API)
- Any OpenAI-compatible local endpoint

Examples:
  armyknife local status
  armyknife local models
  armyknife local chat "Explain this code" --model qwen2.5-coder:3b
  armyknife local generate "Write a function to sort an array"
  armyknife local test --model phi3:mini`,
}

// localStatusCmd checks local Ollama status
var localStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check local Ollama/node-llm status",
	Long:  `Check if Ollama or node-llm is running and accessible.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🔍 Checking local AI status...\n")
		fmt.Printf("   URL: %s\n\n", localOllamaURL)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}

		// Check Ollama API
		resp, err := client.Get(localOllamaURL)
		if err != nil {
			fmt.Printf("❌ Cannot connect to %s\n", localOllamaURL)
			fmt.Printf("   Error: %v\n\n", err)
			fmt.Println("Make sure Ollama is running:")
			fmt.Println("  1. Install: brew install ollama")
			fmt.Println("  2. Start:   ollama serve")
			fmt.Println("  3. Pull:    ollama pull qwen2.5-coder:3b")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("✅ Ollama is running!\n")
			fmt.Printf("   Response: %s\n\n", strings.TrimSpace(string(body)))

			// Get version
			versionResp, err := client.Get(localOllamaURL + "/api/version")
			if err == nil {
				defer versionResp.Body.Close()
				var version map[string]interface{}
				if json.NewDecoder(versionResp.Body).Decode(&version) == nil {
					fmt.Printf("   Version: %v\n", version["version"])
				}
			}

			// List models
			fmt.Println("\n📦 Installed Models:")
			modelsResp, err := client.Get(localOllamaURL + "/api/tags")
			if err == nil {
				defer modelsResp.Body.Close()
				var models map[string]interface{}
				if json.NewDecoder(modelsResp.Body).Decode(&models) == nil {
					if modelList, ok := models["models"].([]interface{}); ok {
						if len(modelList) == 0 {
							fmt.Println("   No models installed. Pull one with: ollama pull qwen2.5-coder:3b")
						}
						for _, m := range modelList {
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
				}
			}
		} else {
			fmt.Printf("⚠️  Unexpected response: %d\n", resp.StatusCode)
		}
	},
}

// localModelsCmd lists available models
var localModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available local models",
	Long:  `List all models available in the local Ollama instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("📦 Local Models (%s)\n", localOllamaURL)
		fmt.Println(strings.Repeat("-", 50))

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Get(localOllamaURL + "/api/tags")
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
				fmt.Println("\nRecommended models for code:")
				fmt.Println("  ollama pull qwen2.5-coder:3b     # Fast, good for completions")
				fmt.Println("  ollama pull qwen2.5-coder:7b     # Better quality")
				fmt.Println("  ollama pull deepseek-coder:6.7b  # Strong coding model")
				fmt.Println("  ollama pull codellama:7b         # Meta's code model")
				fmt.Println("  ollama pull phi3:mini            # Microsoft's small model")
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

				modifiedAt := ""
				if modified, ok := model["modified_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339Nano, modified); err == nil {
						modifiedAt = t.Format("2006-01-02")
					}
				}

				fmt.Printf("  %-30s %10s   %s\n", name, sizeStr, modifiedAt)
			}
		}
	},
}

// localChatCmd sends a chat message
var localChatCmd = &cobra.Command{
	Use:   "chat <message>",
	Short: "Chat with local AI model",
	Long: `Send a chat message to the local AI model.

Examples:
  armyknife local chat "Explain this Go code"
  armyknife local chat "How do I implement a binary tree?" --model codellama:7b
  armyknife local chat "Review this function for bugs" --stream`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message := args[0]

		fmt.Printf("💬 Chat with %s\n", localModel)
		fmt.Println(strings.Repeat("-", 50))

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
			localOllamaURL+"/api/chat",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if localStream {
			// Handle streaming response
			decoder := json.NewDecoder(resp.Body)
			for {
				var chunk map[string]interface{}
				if err := decoder.Decode(&chunk); err != nil {
					break
				}
				if message, ok := chunk["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						fmt.Print(content)
					}
				}
				if done, ok := chunk["done"].(bool); ok && done {
					break
				}
			}
			fmt.Println()
		} else {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("❌ Error parsing response: %v\n", err)
				return
			}

			if message, ok := result["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					fmt.Println(content)
				}
			}

			// Show timing info
			if totalDuration, ok := result["total_duration"].(float64); ok {
				fmt.Printf("\n⏱️  Total: %.2fs", totalDuration/1e9)
				if evalCount, ok := result["eval_count"].(float64); ok {
					tokensPerSec := evalCount / (totalDuration / 1e9)
					fmt.Printf(" | %.1f tokens/sec", tokensPerSec)
				}
				fmt.Println()
			}
		}
	},
}

// localGenerateCmd generates text
var localGenerateCmd = &cobra.Command{
	Use:   "generate <prompt>",
	Short: "Generate text with local AI",
	Long: `Generate text completion from a prompt.

Examples:
  armyknife local generate "// Function to calculate fibonacci"
  armyknife local generate "func sortSlice(s []int) []int {"
  armyknife local generate "Write unit tests for:" --model qwen2.5-coder:7b`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompt := args[0]

		fmt.Printf("🤖 Generating with %s...\n\n", localModel)

		reqBody := map[string]interface{}{
			"model":  localModel,
			"prompt": prompt,
			"stream": localStream,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			localOllamaURL+"/api/generate",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if localStream {
			decoder := json.NewDecoder(resp.Body)
			for {
				var chunk map[string]interface{}
				if err := decoder.Decode(&chunk); err != nil {
					break
				}
				if response, ok := chunk["response"].(string); ok {
					fmt.Print(response)
				}
				if done, ok := chunk["done"].(bool); ok && done {
					break
				}
			}
			fmt.Println()
		} else {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				return
			}

			if response, ok := result["response"].(string); ok {
				fmt.Println(response)
			}

			if totalDuration, ok := result["total_duration"].(float64); ok {
				fmt.Printf("\n⏱️  %.2fs", totalDuration/1e9)
				if evalCount, ok := result["eval_count"].(float64); ok {
					fmt.Printf(" | %.1f tokens/sec", evalCount/(totalDuration/1e9))
				}
				fmt.Println()
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
3. Bug detection
4. Documentation generation`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🧪 Testing local model: %s\n", localModel)
		fmt.Printf("   URL: %s\n", localOllamaURL)
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
				"model":  localModel,
				"prompt": test.prompt,
				"stream": false,
				"options": map[string]interface{}{
					"num_predict": 150,
				},
			}

			jsonData, _ := json.Marshal(reqBody)
			start := time.Now()

			resp, err := client.Post(
				localOllamaURL+"/api/generate",
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

			if response, ok := result["response"].(string); ok {
				// Truncate long responses
				if len(response) > 300 {
					response = response[:300] + "..."
				}
				fmt.Println(response)
			}

			if duration, ok := result["total_duration"].(float64); ok {
				totalTime += duration / 1e9
			}
			if tokens, ok := result["eval_count"].(float64); ok {
				totalTokens += tokens
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

// localPullCmd pulls a model
var localPullCmd = &cobra.Command{
	Use:   "pull <model>",
	Short: "Pull a model from Ollama registry",
	Long: `Download a model from the Ollama registry.

Recommended models for code:
  qwen2.5-coder:3b     - Fast, good for completions (2GB)
  qwen2.5-coder:7b     - Better quality (4GB)
  deepseek-coder:6.7b  - Strong coding model (4GB)
  codellama:7b         - Meta's code model (4GB)
  phi3:mini            - Microsoft's small model (2GB)
  llama3.2:3b          - Meta's latest small model (2GB)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		model := args[0]

		fmt.Printf("📥 Pulling model: %s\n", model)
		fmt.Printf("   This may take a while...\n\n")

		reqBody := map[string]interface{}{
			"name":   model,
			"stream": true,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: 30 * time.Minute} // Long timeout for downloads
		resp, err := client.Post(
			localOllamaURL+"/api/pull",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		lastStatus := ""
		for {
			var chunk map[string]interface{}
			if err := decoder.Decode(&chunk); err != nil {
				break
			}

			status := ""
			if s, ok := chunk["status"].(string); ok {
				status = s
			}

			// Show progress
			if status != lastStatus {
				if status == "success" {
					fmt.Printf("\n✅ Successfully pulled %s\n", model)
				} else {
					fmt.Printf("\r%s", status)
				}
				lastStatus = status
			}

			if completed, ok := chunk["completed"].(float64); ok {
				if total, ok := chunk["total"].(float64); ok && total > 0 {
					pct := (completed / total) * 100
					fmt.Printf("\r%s: %.1f%%", status, pct)
				}
			}
		}
		fmt.Println()
	},
}

// localEmbedCmd generates embeddings
var localEmbedCmd = &cobra.Command{
	Use:   "embed <text>",
	Short: "Generate embeddings with local model",
	Long: `Generate vector embeddings for text using a local model.

Examples:
  armyknife local embed "function to sort array"
  armyknife local embed "authentication middleware" --model nomic-embed-text`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		text := args[0]

		fmt.Printf("🧮 Generating embedding with %s\n", localModel)

		reqBody := map[string]interface{}{
			"model":  localModel,
			"prompt": text,
		}

		jsonData, _ := json.Marshal(reqBody)

		client := &http.Client{Timeout: time.Duration(localTimeout) * time.Second}
		resp, err := client.Post(
			localOllamaURL+"/api/embeddings",
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

		if embedding, ok := result["embedding"].([]interface{}); ok {
			fmt.Printf("✅ Generated embedding\n")
			fmt.Printf("   Dimensions: %d\n", len(embedding))
			if len(embedding) >= 3 {
				fmt.Printf("   Preview: [%.4f, %.4f, %.4f, ...]\n",
					embedding[0], embedding[1], embedding[2])
			}
		} else {
			fmt.Printf("❌ No embedding in response\n")
			body, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(body))
		}
	},
}

// nodeLLMCmd tests node-llm endpoint
var nodeLLMCmd = &cobra.Command{
	Use:   "node-llm",
	Short: "Test node-llm OpenAI-compatible endpoint",
	Long: `Test the node-llm service which provides OpenAI-compatible API.

node-llm runs on the gateway servers and provides:
- /v1/chat/completions (OpenAI compatible)
- /v1/models
- Local model inference`,
	Run: func(cmd *cobra.Command, args []string) {
		nodeLLMURL := os.Getenv("NODE_LLM_URL")
		if nodeLLMURL == "" {
			nodeLLMURL = "http://localhost:3001" // Default node-llm port
		}

		fmt.Printf("🔌 Testing node-llm at %s\n", nodeLLMURL)
		fmt.Println(strings.Repeat("-", 50))

		client := &http.Client{Timeout: 10 * time.Second}

		// Test /v1/models
		fmt.Println("\n1. Checking available models...")
		resp, err := client.Get(nodeLLMURL + "/v1/models")
		if err != nil {
			fmt.Printf("❌ Cannot connect: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var models map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&models); err == nil {
			if data, ok := models["data"].([]interface{}); ok {
				fmt.Printf("✅ Found %d models:\n", len(data))
				for _, m := range data {
					if model, ok := m.(map[string]interface{}); ok {
						fmt.Printf("   - %s\n", model["id"])
					}
				}
			}
		}

		// Test chat completion
		fmt.Println("\n2. Testing chat completion...")
		chatReq := map[string]interface{}{
			"model": localModel,
			"messages": []map[string]string{
				{"role": "user", "content": "Say 'Hello from node-llm!' in exactly 5 words."},
			},
			"max_tokens": 20,
		}

		jsonData, _ := json.Marshal(chatReq)
		chatResp, err := client.Post(
			nodeLLMURL+"/v1/chat/completions",
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		if err != nil {
			fmt.Printf("❌ Chat error: %v\n", err)
			return
		}
		defer chatResp.Body.Close()

		var chatResult map[string]interface{}
		if err := json.NewDecoder(chatResp.Body).Decode(&chatResult); err == nil {
			if choices, ok := chatResult["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						fmt.Printf("✅ Response: %s\n", message["content"])
					}
				}
			}
			if usage, ok := chatResult["usage"].(map[string]interface{}); ok {
				fmt.Printf("   Tokens: %v prompt, %v completion\n",
					usage["prompt_tokens"], usage["completion_tokens"])
			}
		}

		fmt.Println("\n✅ node-llm is working!")
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
	localCmd.AddCommand(localPullCmd)
	localCmd.AddCommand(localEmbedCmd)
	localCmd.AddCommand(nodeLLMCmd)

	// Global flags for local commands
	localCmd.PersistentFlags().StringVar(&localOllamaURL, "ollama-url", "http://localhost:11434", "Ollama API URL")
	localCmd.PersistentFlags().StringVar(&localModel, "model", "qwen2.5-coder:3b", "Model to use")
	localCmd.PersistentFlags().BoolVar(&localStream, "stream", true, "Stream responses")
	localCmd.PersistentFlags().IntVar(&localTimeout, "timeout", 120, "Request timeout in seconds")
}
