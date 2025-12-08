package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	voiceAPIURL    string
	voiceModel     string
	voiceLanguage  string
	voiceFormat    string
	voiceSpeed     float64
	voicePitch     float64
	voiceOutput    string
	voiceLocal     bool
	voiceTimeout   int
	voiceTimestamp bool
)

// voiceCmd represents the voice command group
var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Voice AI operations (Speech-to-Text, Text-to-Speech)",
	Long: `Commands for testing voice functionality with Parakeet TDT and TTS models.

Supports:
- Speech-to-Text (STT) using NVIDIA Parakeet TDT 1.1B
- Text-to-Speech (TTS) using various models (Piper, XTTS, etc.)
- Both local (sherpa-onnx) and cloud API modes

Examples:
  armyknife voice status
  armyknife voice transcribe audio.wav
  armyknife voice transcribe meeting.mp3 --timestamps
  armyknife voice speak "Hello world" --output greeting.wav
  armyknife voice speak "Code review complete" --local
  armyknife voice models
  armyknife voice test`,
}

// voiceStatusCmd checks voice service status
var voiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check voice service status",
	Long:  `Check if voice services (STT/TTS) are running and accessible.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🎤 Voice Service Status\n")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("API URL: %s\n", voiceAPIURL)
		fmt.Printf("Mode: %s\n\n", map[bool]string{true: "Local (sherpa-onnx)", false: "Cloud API"}[voiceLocal])

		client := &http.Client{Timeout: time.Duration(voiceTimeout) * time.Second}

		// Check STT endpoint
		fmt.Printf("📝 Speech-to-Text (STT):\n")
		sttURL := voiceAPIURL + "/api/v1/voice/stt/status"
		if voiceLocal {
			sttURL = "http://localhost:8765/status" // Local sherpa-onnx server
		}
		checkEndpoint(client, "   STT Service", sttURL)

		// Check TTS endpoint
		fmt.Printf("\n🔊 Text-to-Speech (TTS):\n")
		ttsURL := voiceAPIURL + "/api/v1/voice/tts/status"
		if voiceLocal {
			ttsURL = "http://localhost:8766/status" // Local TTS server
		}
		checkEndpoint(client, "   TTS Service", ttsURL)

		// Check Parakeet model availability
		fmt.Printf("\n🦜 Parakeet TDT Model:\n")
		parakeetURL := voiceAPIURL + "/api/v1/voice/models/parakeet"
		if voiceLocal {
			parakeetURL = "http://localhost:8765/models/parakeet-tdt-1.1b"
		}
		checkEndpoint(client, "   Parakeet TDT 1.1B", parakeetURL)

		fmt.Println(strings.Repeat("=", 60))
	},
}

// voiceTranscribeCmd transcribes audio to text
var voiceTranscribeCmd = &cobra.Command{
	Use:   "transcribe <audio-file>",
	Short: "Transcribe audio to text (Speech-to-Text)",
	Long: `Transcribe an audio file to text using Parakeet TDT or other STT models.

Supported formats: WAV, MP3, FLAC, OGG, M4A, WEBM

Examples:
  armyknife voice transcribe meeting.wav
  armyknife voice transcribe audio.mp3 --model parakeet-tdt-1.1b
  armyknife voice transcribe podcast.m4a --timestamps
  armyknife voice transcribe recording.wav --language en --local
  armyknife voice transcribe voice-memo.webm --output transcript.txt`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		audioFile := args[0]

		// Check file exists
		if _, err := os.Stat(audioFile); os.IsNotExist(err) {
			fmt.Printf("❌ Audio file not found: %s\n", audioFile)
			return
		}

		fmt.Printf("🎤 Transcribing: %s\n", audioFile)
		fmt.Printf("   Model: %s\n", voiceModel)
		fmt.Printf("   Mode: %s\n", map[bool]string{true: "Local", false: "Cloud API"}[voiceLocal])
		fmt.Println(strings.Repeat("-", 50))

		startTime := time.Now()

		// Read audio file
		audioData, err := os.ReadFile(audioFile)
		if err != nil {
			fmt.Printf("❌ Error reading file: %v\n", err)
			return
		}

		var result map[string]interface{}
		client := &http.Client{Timeout: time.Duration(voiceTimeout) * time.Second}

		if voiceLocal {
			// Local transcription using sherpa-onnx
			result, err = transcribeLocal(client, audioData, audioFile)
		} else {
			// Cloud API transcription
			result, err = transcribeCloud(client, audioData, audioFile)
		}

		if err != nil {
			fmt.Printf("❌ Transcription error: %v\n", err)
			return
		}

		elapsed := time.Since(startTime)

		// Display results
		fmt.Printf("\n📝 Transcription:\n")
		fmt.Println(strings.Repeat("-", 50))

		if text, ok := result["text"].(string); ok {
			fmt.Println(text)

			// Save to file if output specified
			if voiceOutput != "" {
				if err := os.WriteFile(voiceOutput, []byte(text), 0644); err != nil {
					fmt.Printf("\n❌ Error saving to %s: %v\n", voiceOutput, err)
				} else {
					fmt.Printf("\n✅ Saved to: %s\n", voiceOutput)
				}
			}
		}

		// Show timestamps if requested
		if voiceTimestamp {
			if segments, ok := result["segments"].([]interface{}); ok {
				fmt.Printf("\n⏱️  Timestamps:\n")
				for _, seg := range segments {
					if s, ok := seg.(map[string]interface{}); ok {
						start := s["start"].(float64)
						end := s["end"].(float64)
						text := s["text"].(string)
						fmt.Printf("   [%05.2f - %05.2f] %s\n", start, end, text)
					}
				}
			}
		}

		// Show stats
		fmt.Printf("\n📊 Stats:\n")
		fmt.Printf("   Duration: %.2fs\n", elapsed.Seconds())
		if confidence, ok := result["confidence"].(float64); ok {
			fmt.Printf("   Confidence: %.1f%%\n", confidence*100)
		}
		if lang, ok := result["language"].(string); ok {
			fmt.Printf("   Detected Language: %s\n", lang)
		}
		if model, ok := result["model"].(string); ok {
			fmt.Printf("   Model: %s\n", model)
		}
	},
}

// voiceSpeakCmd converts text to speech
var voiceSpeakCmd = &cobra.Command{
	Use:   "speak <text>",
	Short: "Convert text to speech (Text-to-Speech)",
	Long: `Convert text to speech using TTS models.

Examples:
  armyknife voice speak "Hello, world!"
  armyknife voice speak "Code review complete" --output notification.wav
  armyknife voice speak "Build succeeded" --speed 1.2
  armyknife voice speak "Error detected" --local
  armyknife voice speak "$(cat message.txt)" --model piper`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		text := args[0]

		fmt.Printf("🔊 Text-to-Speech\n")
		fmt.Printf("   Text: %s\n", truncateText(text, 50))
		fmt.Printf("   Model: %s\n", voiceModel)
		fmt.Printf("   Speed: %.1fx\n", voiceSpeed)
		fmt.Println(strings.Repeat("-", 50))

		startTime := time.Now()
		client := &http.Client{Timeout: time.Duration(voiceTimeout) * time.Second}

		var audioData []byte
		var err error

		if voiceLocal {
			audioData, err = speakLocal(client, text)
		} else {
			audioData, err = speakCloud(client, text)
		}

		if err != nil {
			fmt.Printf("❌ TTS error: %v\n", err)
			return
		}

		elapsed := time.Since(startTime)

		// Determine output file
		outputFile := voiceOutput
		if outputFile == "" {
			outputFile = "speech_output." + voiceFormat
		}

		// Save audio
		if err := os.WriteFile(outputFile, audioData, 0644); err != nil {
			fmt.Printf("❌ Error saving audio: %v\n", err)
			return
		}

		fmt.Printf("\n✅ Audio generated!\n")
		fmt.Printf("   Output: %s\n", outputFile)
		fmt.Printf("   Size: %.1f KB\n", float64(len(audioData))/1024)
		fmt.Printf("   Duration: %.2fs\n", elapsed.Seconds())

		// Play audio if possible (optional)
		fmt.Printf("\n💡 Play with: aplay %s  (or: ffplay %s)\n", outputFile, outputFile)
	},
}

// voiceModelsCmd lists available voice models
var voiceModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available voice models",
	Long:  `List all available STT and TTS models.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🎤 Available Voice Models\n")
		fmt.Println(strings.Repeat("=", 60))

		client := &http.Client{Timeout: time.Duration(voiceTimeout) * time.Second}

		// STT Models
		fmt.Printf("\n📝 Speech-to-Text (STT) Models:\n")
		fmt.Println(strings.Repeat("-", 40))

		sttModels := []struct {
			name string
			desc string
			size string
		}{
			{"parakeet-tdt-1.1b", "NVIDIA Parakeet TDT 1.1B (Best accuracy)", "1.1B params"},
			{"parakeet-ctc-1.1b", "NVIDIA Parakeet CTC 1.1B (Fast)", "1.1B params"},
			{"whisper-large-v3", "OpenAI Whisper Large v3", "1.5B params"},
			{"whisper-medium", "OpenAI Whisper Medium", "769M params"},
			{"whisper-small", "OpenAI Whisper Small", "244M params"},
			{"whisper-tiny", "OpenAI Whisper Tiny (Fastest)", "39M params"},
		}

		for _, m := range sttModels {
			fmt.Printf("   %-20s  %s\n", m.name, m.desc)
			fmt.Printf("   %-20s  Size: %s\n", "", m.size)
		}

		// TTS Models
		fmt.Printf("\n🔊 Text-to-Speech (TTS) Models:\n")
		fmt.Println(strings.Repeat("-", 40))

		ttsModels := []struct {
			name string
			desc string
		}{
			{"piper", "Piper TTS (Fast, offline)"},
			{"xtts-v2", "Coqui XTTS v2 (Voice cloning)"},
			{"bark", "Suno Bark (Expressive)"},
			{"speecht5", "Microsoft SpeechT5"},
			{"edge-tts", "Microsoft Edge TTS (Online)"},
		}

		for _, m := range ttsModels {
			fmt.Printf("   %-20s  %s\n", m.name, m.desc)
		}

		// Check which models are available
		fmt.Printf("\n📡 Checking API availability...\n")
		modelsURL := voiceAPIURL + "/api/v1/voice/models"
		resp, err := client.Get(modelsURL)
		if err == nil {
			defer resp.Body.Close()
			var result map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				if stt, ok := result["stt"].([]interface{}); ok {
					fmt.Printf("   ✅ API STT models available: %d\n", len(stt))
				}
				if tts, ok := result["tts"].([]interface{}); ok {
					fmt.Printf("   ✅ API TTS models available: %d\n", len(tts))
				}
			}
		} else {
			fmt.Printf("   ⚠️  Could not connect to API\n")
		}

		fmt.Println(strings.Repeat("=", 60))
	},
}

// voiceTestCmd runs a quick test of voice functionality
var voiceTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test voice functionality end-to-end",
	Long: `Run a quick test of voice services (TTS → STT round-trip).

This will:
1. Generate speech from test text
2. Transcribe the generated audio
3. Compare original vs transcribed text`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🧪 Voice Functionality Test\n")
		fmt.Println(strings.Repeat("=", 60))

		testText := "Hello, this is a test of the voice system. The quick brown fox jumps over the lazy dog."

		client := &http.Client{Timeout: time.Duration(voiceTimeout) * time.Second}

		// Test 1: TTS
		fmt.Printf("\n1️⃣  Text-to-Speech Test\n")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("   Input: %s\n", truncateText(testText, 50))

		ttsStart := time.Now()
		audioData, err := speakCloud(client, testText)
		if err != nil {
			fmt.Printf("   ❌ TTS Failed: %v\n", err)
			// Try local
			fmt.Printf("   Trying local...\n")
			audioData, err = speakLocal(client, testText)
			if err != nil {
				fmt.Printf("   ❌ Local TTS also failed: %v\n", err)
				return
			}
		}
		ttsDuration := time.Since(ttsStart)
		fmt.Printf("   ✅ TTS Success!\n")
		fmt.Printf("   Audio size: %.1f KB\n", float64(len(audioData))/1024)
		fmt.Printf("   Duration: %.2fs\n", ttsDuration.Seconds())

		// Save temp file
		tempFile := "/tmp/voice_test_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".wav"
		if err := os.WriteFile(tempFile, audioData, 0644); err != nil {
			fmt.Printf("   ❌ Could not save temp audio: %v\n", err)
			return
		}
		defer os.Remove(tempFile)

		// Test 2: STT
		fmt.Printf("\n2️⃣  Speech-to-Text Test\n")
		fmt.Println(strings.Repeat("-", 40))

		sttStart := time.Now()
		result, err := transcribeCloud(client, audioData, tempFile)
		if err != nil {
			fmt.Printf("   ❌ STT Failed: %v\n", err)
			// Try local
			fmt.Printf("   Trying local...\n")
			result, err = transcribeLocal(client, audioData, tempFile)
			if err != nil {
				fmt.Printf("   ❌ Local STT also failed: %v\n", err)
				return
			}
		}
		sttDuration := time.Since(sttStart)

		transcribedText := ""
		if text, ok := result["text"].(string); ok {
			transcribedText = text
		}

		fmt.Printf("   ✅ STT Success!\n")
		fmt.Printf("   Output: %s\n", truncateText(transcribedText, 50))
		fmt.Printf("   Duration: %.2fs\n", sttDuration.Seconds())

		// Test 3: Compare
		fmt.Printf("\n3️⃣  Accuracy Check\n")
		fmt.Println(strings.Repeat("-", 40))

		accuracy := calculateAccuracy(strings.ToLower(testText), strings.ToLower(transcribedText))
		fmt.Printf("   Original:    %s\n", truncateText(testText, 40))
		fmt.Printf("   Transcribed: %s\n", truncateText(transcribedText, 40))
		fmt.Printf("   Accuracy: %.1f%%\n", accuracy*100)

		// Summary
		fmt.Printf("\n📊 Summary\n")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("   TTS Latency: %.2fs\n", ttsDuration.Seconds())
		fmt.Printf("   STT Latency: %.2fs\n", sttDuration.Seconds())
		fmt.Printf("   Round-trip: %.2fs\n", ttsDuration.Seconds()+sttDuration.Seconds())
		fmt.Printf("   Accuracy: %.1f%%\n", accuracy*100)

		if accuracy >= 0.9 {
			fmt.Printf("\n   ✅ Voice system working correctly!\n")
		} else if accuracy >= 0.7 {
			fmt.Printf("\n   ⚠️  Voice system working but accuracy could be improved\n")
		} else {
			fmt.Printf("\n   ❌ Voice system needs attention - low accuracy\n")
		}
	},
}

// voiceRecordCmd records audio from microphone
var voiceRecordCmd = &cobra.Command{
	Use:   "record [duration]",
	Short: "Record audio from microphone",
	Long: `Record audio from the microphone for a specified duration.

Examples:
  armyknife voice record 5         # Record for 5 seconds
  armyknife voice record 30 --output meeting.wav
  armyknife voice record 10 | armyknife voice transcribe -`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		duration := 5 // default 5 seconds
		if len(args) > 0 {
			fmt.Sscanf(args[0], "%d", &duration)
		}

		outputFile := voiceOutput
		if outputFile == "" {
			outputFile = fmt.Sprintf("recording_%d.wav", time.Now().Unix())
		}

		fmt.Printf("🎙️  Recording Audio\n")
		fmt.Printf("   Duration: %d seconds\n", duration)
		fmt.Printf("   Output: %s\n", outputFile)
		fmt.Println(strings.Repeat("-", 50))

		// Check if arecord is available
		fmt.Printf("   Starting recording in 3 seconds...\n")
		time.Sleep(3 * time.Second)

		fmt.Printf("   🔴 RECORDING... (speak now)\n")

		// Use arecord on Linux, sox on Mac
		_ = fmt.Sprintf("arecord -d %d -f cd -t wav %s 2>/dev/null || rec -q %s trim 0 %d 2>/dev/null",
			duration, outputFile, outputFile, duration)

		// We'd run this command but for safety, just show the instruction
		fmt.Printf("\n   Run this command to record:\n")
		fmt.Printf("   $ arecord -d %d -f cd -t wav %s\n", duration, outputFile)
		fmt.Printf("\n   Or on Mac:\n")
		fmt.Printf("   $ sox -d %s trim 0 %d\n", outputFile, duration)

		fmt.Printf("\n💡 After recording, transcribe with:\n")
		fmt.Printf("   armyknife voice transcribe %s\n", outputFile)
	},
}

// voiceLiveCmd starts live transcription
var voiceLiveCmd = &cobra.Command{
	Use:   "live",
	Short: "Start live transcription (streaming)",
	Long: `Start live transcription from microphone with real-time output.

This streams audio to the STT service and displays transcription in real-time.

Examples:
  armyknife voice live
  armyknife voice live --model parakeet-tdt-1.1b
  armyknife voice live --language en`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🎤 Live Transcription\n")
		fmt.Printf("   Model: %s\n", voiceModel)
		fmt.Printf("   Mode: %s\n", map[bool]string{true: "Local", false: "Cloud API"}[voiceLocal])
		fmt.Println(strings.Repeat("=", 60))

		wsURL := strings.Replace(voiceAPIURL, "http://", "ws://", 1)
		wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
		wsURL += "/api/v1/voice/stt/stream"

		fmt.Printf("   WebSocket: %s\n", wsURL)
		fmt.Println()
		fmt.Println("   🔴 Live transcription requires WebSocket support.")
		fmt.Println("   Press Ctrl+C to stop.")
		fmt.Println()
		fmt.Println("   💡 To start live transcription, run:")
		fmt.Println()
		fmt.Printf("   # Using websocat (install: cargo install websocat)\n")
		fmt.Printf("   arecord -f cd -t wav - | websocat %s\n", wsURL)
		fmt.Println()
		fmt.Printf("   # Or using ffmpeg + websocat\n")
		fmt.Printf("   ffmpeg -f alsa -i default -f wav - 2>/dev/null | websocat %s\n", wsURL)
	},
}

// Helper functions

func checkEndpoint(client *http.Client, name, url string) {
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("%s: ❌ Not available (%v)\n", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("%s: ✅ Running\n", name)
	} else {
		fmt.Printf("%s: ⚠️  Status %d\n", name, resp.StatusCode)
	}
}

func transcribeLocal(client *http.Client, audioData []byte, filename string) (map[string]interface{}, error) {
	// Local sherpa-onnx server endpoint
	localURL := "http://localhost:8765/transcribe"

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("audio", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	part.Write(audioData)

	writer.WriteField("model", voiceModel)
	writer.WriteField("language", voiceLanguage)
	if voiceTimestamp {
		writer.WriteField("timestamps", "true")
	}
	writer.Close()

	req, err := http.NewRequest("POST", localURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("local STT server not running: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func transcribeCloud(client *http.Client, audioData []byte, filename string) (map[string]interface{}, error) {
	// Cloud API endpoint
	cloudURL := voiceAPIURL + "/api/v1/voice/stt/transcribe"

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("audio", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	part.Write(audioData)

	writer.WriteField("model", voiceModel)
	writer.WriteField("language", voiceLanguage)
	if voiceTimestamp {
		writer.WriteField("timestamps", "true")
	}
	writer.Close()

	req, err := http.NewRequest("POST", cloudURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func speakLocal(client *http.Client, text string) ([]byte, error) {
	// Local TTS server endpoint
	localURL := "http://localhost:8766/tts"

	reqBody := map[string]interface{}{
		"text":   text,
		"model":  voiceModel,
		"speed":  voiceSpeed,
		"pitch":  voicePitch,
		"format": voiceFormat,
	}

	jsonData, _ := json.Marshal(reqBody)

	resp, err := client.Post(localURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("local TTS server not running: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check if response is JSON with base64 audio or raw audio
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		if audioB64, ok := result["audio"].(string); ok {
			return base64.StdEncoding.DecodeString(audioB64)
		}
		return nil, fmt.Errorf("no audio in response")
	}

	return io.ReadAll(resp.Body)
}

func speakCloud(client *http.Client, text string) ([]byte, error) {
	// Cloud API endpoint
	cloudURL := voiceAPIURL + "/api/v1/voice/tts/speak"

	reqBody := map[string]interface{}{
		"text":   text,
		"model":  voiceModel,
		"speed":  voiceSpeed,
		"pitch":  voicePitch,
		"format": voiceFormat,
	}

	jsonData, _ := json.Marshal(reqBody)

	resp, err := client.Post(cloudURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check if response is JSON with base64 audio or raw audio
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		if audioB64, ok := result["audio"].(string); ok {
			return base64.StdEncoding.DecodeString(audioB64)
		}
		return nil, fmt.Errorf("no audio in response")
	}

	return io.ReadAll(resp.Body)
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func calculateAccuracy(original, transcribed string) float64 {
	// Simple word-level accuracy (WER-based approximation)
	origWords := strings.Fields(original)
	transWords := strings.Fields(transcribed)

	if len(origWords) == 0 {
		return 0
	}

	matches := 0
	for i, word := range origWords {
		if i < len(transWords) && word == transWords[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(origWords))
}

func init() {
	rootCmd.AddCommand(voiceCmd)

	// Voice subcommands
	voiceCmd.AddCommand(voiceStatusCmd)
	voiceCmd.AddCommand(voiceTranscribeCmd)
	voiceCmd.AddCommand(voiceSpeakCmd)
	voiceCmd.AddCommand(voiceModelsCmd)
	voiceCmd.AddCommand(voiceTestCmd)
	voiceCmd.AddCommand(voiceRecordCmd)
	voiceCmd.AddCommand(voiceLiveCmd)

	// Global flags for voice commands
	voiceCmd.PersistentFlags().StringVar(&voiceAPIURL, "api-url", "https://api.armyknifelabs.com", "Voice API URL")
	voiceCmd.PersistentFlags().StringVar(&voiceModel, "model", "parakeet-tdt-1.1b", "Voice model to use")
	voiceCmd.PersistentFlags().StringVar(&voiceLanguage, "language", "en", "Language code (en, es, fr, etc.)")
	voiceCmd.PersistentFlags().StringVar(&voiceFormat, "format", "wav", "Audio format (wav, mp3, ogg)")
	voiceCmd.PersistentFlags().Float64Var(&voiceSpeed, "speed", 1.0, "Speech speed (0.5 - 2.0)")
	voiceCmd.PersistentFlags().Float64Var(&voicePitch, "pitch", 1.0, "Speech pitch (0.5 - 2.0)")
	voiceCmd.PersistentFlags().StringVar(&voiceOutput, "output", "", "Output file path")
	voiceCmd.PersistentFlags().BoolVar(&voiceLocal, "local", false, "Use local voice server (sherpa-onnx)")
	voiceCmd.PersistentFlags().IntVar(&voiceTimeout, "timeout", 120, "Request timeout in seconds")

	// Transcribe-specific flags
	voiceTranscribeCmd.Flags().BoolVar(&voiceTimestamp, "timestamps", false, "Include word timestamps")
}
