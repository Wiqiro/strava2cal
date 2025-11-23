package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type StravaToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type AppState struct {
	Token          *StravaToken `json:"token,omitempty"`
	SubscriptionID *int         `json:"subscription_id,omitempty"`
}

func loadState() (*AppState, error) {
	data, err := os.ReadFile(STATE_FILE)
	if err != nil {
		if os.IsNotExist(err) {
			return &AppState{}, nil
		}
		return nil, err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveState(state *AppState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(STATE_FILE, data, 0o600)
}

func LoadToken() (*StravaToken, error) {
	state, err := loadState()
	if err != nil {
		return nil, err
	}
	if state.Token == nil {
		return nil, fmt.Errorf("no token stored")
	}
	return state.Token, nil
}

func SaveToken(token *StravaToken) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	state.Token = token
	return saveState(state)
}

func LoadSubscriptionID() *int {
	state, err := loadState()
	if err != nil {
		return nil
	}
	return state.SubscriptionID
}

func SaveSubscriptionID(id int) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	state.SubscriptionID = &id
	return saveState(state)
}

func ExchangeCode(clientId, clientSecret, code string) (*StravaToken, error) {
	url := "https://www.strava.com/oauth/token"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
	q.Add("code", code)
	q.Add("grant_type", "authorization_code")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to exchange code, status code: %d", resp.StatusCode)
	}

	var token StravaToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func RefreshToken(clientId, clientSecret, refreshToken string) (*StravaToken, error) {
	url := "https://www.strava.com/oauth/token"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
	q.Add("grant_type", "refresh_token")
	q.Add("refresh_token", refreshToken)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to refresh token, status code: %d", resp.StatusCode)
	}

	var token StravaToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}
