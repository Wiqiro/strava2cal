package main

import (
	"encoding/json"
	"os"
)

type AppState struct {
	Token          *StravaToken `json:"token,omitempty"`
	SubscriptionId int          `json:"subscription_id,omitempty"`
	Activities     []Activity   `json:"activities"`
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

func loadSubscriptionId() (int, error) {
	state, err := loadState()
	if err != nil {
		return 0, err
	}
	return state.SubscriptionId, nil
}

func saveSubscriptionID(id int) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	state.SubscriptionId = id
	return saveState(state)
}
