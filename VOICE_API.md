# Voice API - CLI to Endpoint Mapping

This document maps armyknife-cli voice commands to their corresponding API endpoints.

## Overview

The Voice API provides Speech-to-Text (STT) and Text-to-Speech (TTS) capabilities, with support for both cloud API and local (sherpa-onnx) modes.

### Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                    ArmyKnife CLI (Voice Commands)                  │
├────────────────────────────────────────────────────────────────────┤
│  armyknife voice transcribe    armyknife voice speak               │
│  armyknife voice status        armyknife voice models              │
│  armyknife voice test          armyknife voice live                │
└──────────────────────┬────────────────────────────┬────────────────┘
                       │                            │
              ┌────────▼────────┐         ┌────────▼────────┐
              │   Cloud API     │         │   Local Mode    │
              │ (API Gateway)   │         │ (sherpa-onnx)   │
              └────────┬────────┘         └────────┬────────┘
                       │                            │
       ┌───────────────┼───────────────┐           │
       │               │               │           │
┌──────▼─────┐  ┌──────▼─────┐  ┌──────▼─────┐    │
│  Parakeet  │  │  Whisper   │  │   Piper    │    │
│  TDT 1.1B  │  │  Large v3  │  │    TTS     │    │
└────────────┘  └────────────┘  └────────────┘    │
                                                   │
                                         ┌─────────▼─────────┐
                                         │   sherpa-onnx     │
                                         │  (offline ASR)    │
                                         └───────────────────┘
```

## CLI Commands → API Endpoints

### 1. Voice Status

**CLI Command:**
```bash
armyknife voice status
armyknife voice status --local
```

**Cloud API Endpoints:**
```
GET /api/v1/voice/stt/status
GET /api/v1/voice/tts/status
GET /api/v1/voice/models/parakeet
```

**Local Endpoints:**
```
GET http://localhost:8765/status    # Local STT server
GET http://localhost:8766/status    # Local TTS server
```

**Response:**
```json
{
  "service": "stt",
  "status": "running",
  "model": "parakeet-tdt-1.1b",
  "version": "1.0.0",
  "features": ["timestamps", "word_confidence", "language_detection"]
}
```

---

### 2. Voice Transcribe (Speech-to-Text)

**CLI Command:**
```bash
armyknife voice transcribe audio.wav
armyknife voice transcribe audio.mp3 --model parakeet-tdt-1.1b
armyknife voice transcribe meeting.m4a --timestamps
armyknife voice transcribe recording.wav --local --output transcript.txt
```

**Cloud API Endpoint:**
```
POST /api/v1/voice/stt/transcribe
Content-Type: multipart/form-data
```

**Local Endpoint:**
```
POST http://localhost:8765/transcribe
Content-Type: multipart/form-data
```

**Request (multipart/form-data):**
```
audio: <binary audio file>
model: "parakeet-tdt-1.1b"
language: "en"
timestamps: "true"
```

**Response:**
```json
{
  "text": "Hello, this is a test transcription.",
  "confidence": 0.95,
  "language": "en",
  "model": "parakeet-tdt-1.1b",
  "duration_ms": 3500,
  "segments": [
    {
      "start": 0.0,
      "end": 1.2,
      "text": "Hello,",
      "confidence": 0.98
    },
    {
      "start": 1.3,
      "end": 3.5,
      "text": "this is a test transcription.",
      "confidence": 0.93
    }
  ]
}
```

---

### 3. Voice Speak (Text-to-Speech)

**CLI Command:**
```bash
armyknife voice speak "Hello, world!"
armyknife voice speak "Build complete" --output notification.wav
armyknife voice speak "Error detected" --speed 1.2 --model piper
armyknife voice speak "Test message" --local
```

**Cloud API Endpoint:**
```
POST /api/v1/voice/tts/speak
Content-Type: application/json
```

**Local Endpoint:**
```
POST http://localhost:8766/tts
Content-Type: application/json
```

**Request:**
```json
{
  "text": "Hello, world!",
  "model": "piper",
  "speed": 1.0,
  "pitch": 1.0,
  "format": "wav",
  "voice": "en_US-amy-medium"
}
```

**Response (binary audio or base64):**
```json
{
  "audio": "<base64-encoded-audio>",
  "format": "wav",
  "duration_ms": 1200,
  "sample_rate": 22050,
  "model": "piper"
}
```

Or raw audio bytes with headers:
```
Content-Type: audio/wav
Content-Length: 48000
```

---

### 4. Voice Models

**CLI Command:**
```bash
armyknife voice models
```

**Cloud API Endpoint:**
```
GET /api/v1/voice/models
```

**Response:**
```json
{
  "stt": [
    {
      "id": "parakeet-tdt-1.1b",
      "name": "NVIDIA Parakeet TDT 1.1B",
      "description": "State-of-the-art ASR model",
      "languages": ["en"],
      "features": ["timestamps", "word_confidence"],
      "size_mb": 1100
    },
    {
      "id": "whisper-large-v3",
      "name": "OpenAI Whisper Large v3",
      "description": "Multilingual ASR",
      "languages": ["en", "es", "fr", "de", "zh", "ja", "..."],
      "features": ["timestamps", "language_detection", "translation"],
      "size_mb": 1500
    }
  ],
  "tts": [
    {
      "id": "piper",
      "name": "Piper TTS",
      "description": "Fast offline TTS",
      "voices": ["en_US-amy-medium", "en_US-lessac-medium"],
      "features": ["speed", "pitch"]
    },
    {
      "id": "xtts-v2",
      "name": "Coqui XTTS v2",
      "description": "Voice cloning TTS",
      "voices": ["default"],
      "features": ["voice_cloning", "speed"]
    }
  ]
}
```

---

### 5. Voice Test

**CLI Command:**
```bash
armyknife voice test
```

**Cloud API Endpoint (combines TTS + STT):**
```
POST /api/v1/voice/test
Content-Type: application/json
```

**Request:**
```json
{
  "test_text": "Hello, this is a test."
}
```

**Response:**
```json
{
  "success": true,
  "tts": {
    "status": "ok",
    "latency_ms": 450,
    "model": "piper"
  },
  "stt": {
    "status": "ok",
    "latency_ms": 1200,
    "model": "parakeet-tdt-1.1b"
  },
  "round_trip_ms": 1650,
  "original_text": "Hello, this is a test.",
  "transcribed_text": "Hello, this is a test.",
  "accuracy": 1.0
}
```

---

### 6. Voice Live (Streaming STT)

**CLI Command:**
```bash
armyknife voice live
armyknife voice live --model parakeet-tdt-1.1b
```

**WebSocket Endpoint:**
```
WS /api/v1/voice/stt/stream
```

**Connection Flow:**
```
1. Client connects to WebSocket
2. Client sends audio chunks (binary)
3. Server sends partial transcriptions (JSON)
4. Connection stays open until client closes
```

**Incoming (Client → Server):**
```
Binary audio data (PCM 16-bit, 16kHz mono)
```

**Outgoing (Server → Client):**
```json
{
  "type": "partial",
  "text": "Hello this is",
  "is_final": false
}
```

```json
{
  "type": "final",
  "text": "Hello, this is a test.",
  "is_final": true,
  "confidence": 0.95,
  "segments": [...]
}
```

---

## Backend Implementation Requirements

### Files to Create

1. **Routes**: `src/routes/voice/voice.routes.ts`
2. **Service**: `src/services/VoiceService.ts`
3. **STT Service**: `src/services/voice/STTService.ts`
4. **TTS Service**: `src/services/voice/TTSService.ts`

### VoiceService Interface

```typescript
interface VoiceService {
  // Status
  getSTTStatus(): Promise<ServiceStatus>;
  getTTSStatus(): Promise<ServiceStatus>;

  // STT
  transcribe(audio: Buffer, options: TranscribeOptions): Promise<TranscriptionResult>;
  streamTranscribe(audioStream: ReadableStream): AsyncGenerator<PartialTranscription>;

  // TTS
  speak(text: string, options: SpeakOptions): Promise<Buffer>;

  // Models
  listModels(): Promise<VoiceModels>;
  getModelInfo(modelId: string): Promise<ModelInfo>;

  // Test
  runTest(testText?: string): Promise<TestResult>;
}

interface TranscribeOptions {
  model?: string;          // default: parakeet-tdt-1.1b
  language?: string;       // default: en
  timestamps?: boolean;    // default: false
  wordConfidence?: boolean; // default: false
}

interface SpeakOptions {
  model?: string;          // default: piper
  voice?: string;          // default: en_US-amy-medium
  speed?: number;          // 0.5 - 2.0, default: 1.0
  pitch?: number;          // 0.5 - 2.0, default: 1.0
  format?: 'wav' | 'mp3' | 'ogg'; // default: wav
}
```

---

## STT Models Supported

| Model | Description | Size | Languages | Speed |
|-------|-------------|------|-----------|-------|
| **parakeet-tdt-1.1b** | NVIDIA Parakeet TDT (Best accuracy) | 1.1B | en | Medium |
| **parakeet-ctc-1.1b** | NVIDIA Parakeet CTC (Faster) | 1.1B | en | Fast |
| **whisper-large-v3** | OpenAI Whisper Large v3 | 1.5B | 99+ | Slow |
| **whisper-medium** | OpenAI Whisper Medium | 769M | 99+ | Medium |
| **whisper-small** | OpenAI Whisper Small | 244M | 99+ | Fast |
| **whisper-tiny** | OpenAI Whisper Tiny | 39M | 99+ | Very Fast |

### Parakeet TDT 1.1B (Recommended)

- **Architecture**: Transducer-based (TDT)
- **Training Data**: 64,000 hours of English speech
- **Word Error Rate (WER)**: 5.66% on LibriSpeech test-clean
- **Features**: Word timestamps, confidence scores
- **Use Case**: Best for English transcription accuracy

---

## TTS Models Supported

| Model | Description | Features | Speed |
|-------|-------------|----------|-------|
| **piper** | Piper TTS (Fast, offline) | Speed, pitch control | Very Fast |
| **xtts-v2** | Coqui XTTS v2 (Voice cloning) | Voice cloning, multilingual | Medium |
| **bark** | Suno Bark (Expressive) | Music, laughter, emotions | Slow |
| **speecht5** | Microsoft SpeechT5 | Multilingual | Medium |
| **edge-tts** | Microsoft Edge TTS | High quality, online | Fast |

---

## Local Mode (sherpa-onnx)

For offline/air-gapped environments, use local mode:

### Local STT Server (Port 8765)

```bash
# Start local STT server
sherpa-onnx-offline-websocket-server \
  --port=8765 \
  --encoder=parakeet-tdt-1.1b-encoder.onnx \
  --decoder=parakeet-tdt-1.1b-decoder.onnx \
  --joiner=parakeet-tdt-1.1b-joiner.onnx \
  --tokens=tokens.txt
```

### Local TTS Server (Port 8766)

```bash
# Start local TTS server (Piper)
piper-http-server \
  --port=8766 \
  --model=/models/piper/en_US-amy-medium.onnx
```

### Configuration

```bash
# Use local mode with CLI
armyknife voice transcribe audio.wav --local
armyknife voice speak "Hello" --local --output hello.wav

# Or set environment variable
export VOICE_MODE=local
export STT_LOCAL_URL=http://localhost:8765
export TTS_LOCAL_URL=http://localhost:8766
```

---

## Integration with VS Code Fork (armyknife-code)

The VS Code fork should integrate voice for:

1. **Voice Dictation**: Speak code instead of typing
2. **Voice Commands**: "Navigate to function X"
3. **Code Explanation**: "Explain this function" (TTS output)
4. **Meeting Notes**: Transcribe code review discussions

### VS Code Settings

```json
{
  "armyknife.voice.enabled": true,
  "armyknife.voice.sttModel": "parakeet-tdt-1.1b",
  "armyknife.voice.ttsModel": "piper",
  "armyknife.voice.mode": "cloud",  // or "local"
  "armyknife.voice.hotkey": "ctrl+shift+v"
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `VOICE_001` | STT service unavailable |
| `VOICE_002` | TTS service unavailable |
| `VOICE_003` | Unsupported audio format |
| `VOICE_004` | Audio too long (max 10 minutes) |
| `VOICE_005` | Model not available |
| `VOICE_006` | Language not supported |
| `VOICE_007` | Rate limit exceeded |
| `VOICE_008` | Invalid audio data |

---

## Rate Limits

| Endpoint | Rate Limit |
|----------|------------|
| `/api/v1/voice/stt/transcribe` | 60 requests/minute |
| `/api/v1/voice/tts/speak` | 120 requests/minute |
| `/api/v1/voice/stt/stream` | 5 concurrent connections |
| `/api/v1/voice/models` | 300 requests/minute |

---

## Environment Variables

```bash
# API Configuration
VOICE_API_URL=https://api.armyknifelabs.com
VOICE_API_KEY=your_api_key

# Default Models
VOICE_STT_MODEL=parakeet-tdt-1.1b
VOICE_TTS_MODEL=piper

# Local Mode
VOICE_MODE=cloud  # or "local"
STT_LOCAL_URL=http://localhost:8765
TTS_LOCAL_URL=http://localhost:8766

# Timeouts
VOICE_TIMEOUT_MS=120000
```

---

## Example Usage

### Transcribe a Meeting Recording

```bash
# Transcribe with timestamps
armyknife voice transcribe meeting.mp3 --timestamps --output meeting_transcript.txt

# Output:
# [00:00.00 - 00:02.50] Welcome everyone to today's standup.
# [00:02.80 - 00:05.20] Let's start with the backend team.
# ...
```

### Generate Voice Notifications

```bash
# Build notification
armyknife voice speak "Build succeeded for PR 123" --output build_success.wav

# Play with system audio
aplay build_success.wav
```

### Live Coding Dictation

```bash
# Start live transcription
armyknife voice live --model parakeet-tdt-1.1b

# Speak: "function calculate total price items"
# Output: function calculateTotalPrice(items)
```

### End-to-End Test

```bash
# Run voice system test
armyknife voice test

# Output:
# 🧪 Voice Functionality Test
# ==============================================================
#
# 1️⃣  Text-to-Speech Test
# ----------------------------------------
#    Input: Hello, this is a test of the voice system...
#    ✅ TTS Success!
#    Audio size: 24.5 KB
#    Duration: 0.45s
#
# 2️⃣  Speech-to-Text Test
# ----------------------------------------
#    ✅ STT Success!
#    Output: Hello, this is a test of the voice system...
#    Duration: 1.20s
#
# 3️⃣  Accuracy Check
# ----------------------------------------
#    Original:    Hello, this is a test of the voice system...
#    Transcribed: Hello, this is a test of the voice system...
#    Accuracy: 100.0%
#
# 📊 Summary
# ==============================================================
#    TTS Latency: 0.45s
#    STT Latency: 1.20s
#    Round-trip: 1.65s
#    Accuracy: 100.0%
#
#    ✅ Voice system working correctly!
```
