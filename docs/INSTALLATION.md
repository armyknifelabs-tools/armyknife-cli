# Installation Guide

## Quick Install

### Option 1: Download Binary (Recommended)

1. **Visit the [Releases Page](https://github.com/armyknifelabs-platform/armyknife-cli/releases)**

2. **Download the appropriate binary for your platform:**
   - macOS (Intel): `armyknife-*-darwin-amd64`
   - macOS (Apple Silicon): `armyknife-*-darwin-arm64`
   - Linux (amd64): `armyknife-*-linux-amd64`
   - Linux (arm64): `armyknife-*-linux-arm64`
   - Windows: `armyknife-*-windows-amd64.exe`

3. **Verify the checksum** (optional but recommended):
   ```bash
   # Download SHA256SUMS file
   sha256sum -c SHA256SUMS
   ```

4. **Make the binary executable** (Linux/macOS):
   ```bash
   chmod +x armyknife-*
   ```

5. **Move to PATH** (Linux/macOS):
   ```bash
   sudo mv armyknife-* /usr/local/bin/armyknife
   ```

6. **Verify installation:**
   ```bash
   armyknife --version
   ```

### Option 2: Homebrew (macOS/Linux)

```bash
# Add the tap
brew tap armyknifelabs/tap

# Install armyknife
brew install armyknifelabs/tap/armyknife

# Verify
armyknife --version
```

### Option 3: Build from Source

**Prerequisites:**
- Go 1.21 or later
- Git

**Steps:**
```bash
# Clone the repository
git clone https://github.com/armyknifelabs-platform/armyknife-cli
cd armyknife-cli

# Build
go build -o armyknife ./cmd

# Optionally, move to PATH
sudo mv armyknife /usr/local/bin/

# Verify
armyknife --version
```

### Option 4: Docker

```bash
# Run in container
docker run -it armyknifelabs/armyknife:latest code query "your query"

# Or mount your environment
docker run -it \
  -e CODE_API_URL="http://host.docker.internal:3001/api/v1" \
  armyknifelabs/armyknife:latest \
  code metrics
```

## Configuration

### Set API Endpoint

```bash
# For local development (default)
export CODE_API_URL="http://localhost:3001/api/v1"

# For production
export CODE_API_URL="https://code-intelligence.armyknifelabs.com/api/v1"
export CODE_API_KEY="your_api_key_here"
```

### Persistent Configuration

**Create `~/.armyknife/config.yaml`:**
```yaml
api:
  url: "http://localhost:3001/api/v1"
  key: "your_api_key_here"
  timeout: 30s

output:
  format: "table"  # or "json"
  color: true

cache:
  enabled: true
  ttl: 3600
```

## Troubleshooting

### "Command not found: armyknife"

1. **Verify binary is executable:**
   ```bash
   ls -la /usr/local/bin/armyknife
   # Should show: -rwxr-xr-x
   ```

2. **Check PATH:**
   ```bash
   echo $PATH
   # Should include /usr/local/bin
   ```

3. **Reinstall:**
   ```bash
   sudo rm /usr/local/bin/armyknife
   # Download and reinstall
   ```

### "API Connection Failed"

1. **Verify API is running:**
   ```bash
   curl http://localhost:3001/api/v1/health
   ```

2. **Check API URL is correct:**
   ```bash
   echo $CODE_API_URL
   ```

3. **Test connection:**
   ```bash
   curl -X GET "$CODE_API_URL/health"
   ```

### "Permission Denied"

On Linux/macOS, make binary executable:
```bash
chmod +x ./armyknife
```

## Upgrading

### From Homebrew
```bash
brew upgrade armyknifelabs/tap/armyknife
```

### From Binary
Simply download the latest release and replace the old binary.

### From Source
```bash
cd armyknife-cli
git pull origin main
go build -o armyknife ./cmd
sudo mv armyknife /usr/local/bin/
```

## Next Steps

1. **Configure your API endpoint:**
   ```bash
   export CODE_API_URL="http://localhost:3001/api/v1"
   ```

2. **Test the installation:**
   ```bash
   armyknife code metrics
   ```

3. **Read the [Complete Guide](https://gist.github.com/armyknife-tools/05e48e844e51db804f788ab220c4b782)**

## Support

- **Issues**: [GitHub Issues](https://github.com/armyknifelabs-platform/armyknife-cli/issues)
- **Documentation**: [Complete Guide](https://gist.github.com/armyknife-tools/05e48e844e51db804f788ab220c4b782)
