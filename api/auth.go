package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type StravaToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func ExchangeCode(code string) (*StravaToken, error) {
	url := "https://www.strava.com/oauth/token"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("client_id", CLIENT_ID)
	q.Add("client_secret", CLIENT_SECRET)
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

func RefreshToken(refreshToken string) (*StravaToken, error) {
	url := "https://www.strava.com/oauth/token"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("client_id", CLIENT_ID)
	q.Add("client_secret", CLIENT_SECRET)
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

func (token *StravaToken) IsTokenExpired() bool {
	return time.Now().Unix() >= token.ExpiresAt-10
}
