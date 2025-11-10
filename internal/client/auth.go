package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/armyknifelabs-platform/armyknife-cli/internal/config"
	"github.com/armyknifelabs-platform/armyknife-cli/pkg/output"
)

// DeviceCodeResponse represents the response from initiating device flow
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error,omitempty"`
}

// AuthenticateDeviceFlow performs OAuth device flow authentication
func AuthenticateDeviceFlow(cfg *config.Config) error {
	output.Info("Initiating GitHub OAuth Device Flow...")

	// Step 1: Request device code
	deviceResp, err := requestDeviceCode(cfg.APIURL)
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}

	// Step 2: Display user code and URL
	output.Success(fmt.Sprintf("\n🔐 Please visit: %s", deviceResp.VerificationURI))
	output.Success(fmt.Sprintf("📋 Enter code: %s\n", deviceResp.UserCode))

	// Step 3: Poll for token
	token, err := pollForToken(cfg.APIURL, deviceResp.DeviceCode, deviceResp.Interval, deviceResp.ExpiresIn)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Step 4: Save token to config
	cfg.AccessToken = token.AccessToken
	cfg.RefreshToken = token.RefreshToken
	cfg.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	output.Success("\n✅ Authentication successful!")
	return nil
}

// requestDeviceCode requests a device code from the API
func requestDeviceCode(apiURL string) (*DeviceCodeResponse, error) {
	url := fmt.Sprintf("%s/auth/github/device/code", apiURL)

	// Send empty JSON object to satisfy Fastify content-type requirement
	emptyBody := bytes.NewBuffer([]byte("{}"))
	req, err := http.NewRequest("POST", url, emptyBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, err
	}

	return &deviceResp, nil
}

// pollForToken polls the token endpoint until authorization is complete
func pollForToken(apiURL, deviceCode string, interval, expiresIn int) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/auth/github/device/token", apiURL)
	pollInterval := time.Duration(interval) * time.Second
	timeout := time.Now().Add(time.Duration(expiresIn) * time.Second)

	output.Info("Waiting for authorization...")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(timeout) {
				return nil, fmt.Errorf("authorization timeout exceeded")
			}

			token, err := checkToken(url, deviceCode)
			if err != nil {
				// Check if it's a retriable error
				if err.Error() == "authorization_pending" || err.Error() == "slow_down" {
					continue
				}
				return nil, err
			}

			if token != nil {
				return token, nil
			}
		}
	}
}

// checkToken checks if the token is available
func checkToken(url, deviceCode string) (*TokenResponse, error) {
	reqBody := map[string]string{
		"device_code": deviceCode,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	// Check for errors
	if tokenResp.Error != "" {
		if tokenResp.Error == "authorization_pending" || tokenResp.Error == "slow_down" {
			return nil, fmt.Errorf(tokenResp.Error)
		}
		return nil, fmt.Errorf("token error: %s", tokenResp.Error)
	}

	return &tokenResp, nil
}
