package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func registerWebhook(callbackUrl, verifyToken string) error {
	webhookUrl := "https://www.strava.com/api/v3/push_subscriptions"

	req, err := http.NewRequest("POST", webhookUrl, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("client_id", CLIENT_ID)
	q.Add("client_secret", CLIENT_SECRET)
	q.Add("callback_url", callbackUrl)
	q.Add("verify_token", verifyToken)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var content struct {
		Id int `json:"id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&content)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register webhook, status code: %d", resp.StatusCode)
	}
	return nil
}

func getWebhook() (int, error) {
	webhookUrl := "https://www.strava.com/api/v3/push_subscriptions"

	req, err := http.NewRequest("GET", webhookUrl, nil)
	if err != nil {
		return 0, err
	}

	q := req.URL.Query()
	q.Add("client_id", CLIENT_ID)
	q.Add("client_secret", CLIENT_SECRET)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var content []struct {
		Id int `json:"id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&content)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get webhook, status code: %d", resp.StatusCode)
	}
	if len(content) == 0 {
		return 0, nil
	}
	return content[0].Id, nil
}

func unregisterWebhook(subscriptionId int) error {
	webhookUrl := fmt.Sprintf("https://www.strava.com/api/v3/push_subscriptions/%d", subscriptionId)

	req, err := http.NewRequest("DELETE", webhookUrl, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("client_id", CLIENT_ID)
	q.Add("client_secret", CLIENT_SECRET)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to unregister webhook, status code: %d", resp.StatusCode)
	}
	return nil
}
