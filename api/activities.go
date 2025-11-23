package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Activity struct {
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Distance    float64 `json:"distance"`
	Type        string  `json:"type"`
	StartDate   string  `json:"start_date"`
	Timezone    string  `json:"timezone"`
	AvgSpeed    float32 `json:"average_speed"`
	AvgWatts    float32 `json:"average_watts"`
	Description string  `json:"description"`
}

func FetchActivity(accessToken string, activityId int) (*Activity, error) {
	url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d", activityId)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch activity, status code: %d", resp.StatusCode)
	}

	var activity Activity
	err = json.NewDecoder(resp.Body).Decode(&activity)
	if err != nil {
		return nil, err
	}

	return &activity, nil
}
