package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BaseActivity struct {
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Distance    float32 `json:"distance"`
	Type        string  `json:"type"`
	Timezone    string  `json:"timezone"`
	AvgSpeed    float32 `json:"average_speed"`
	AvgWatts    float32 `json:"average_watts"`
	AvgCadence  float32 `json:"average_cadence"`
	Description string  `json:"description"`
}

type RawActivity struct {
	BaseActivity
	StartDate   string `json:"start_date"`
	ElapsedTime int    `json:"elapsed_time"`
}

type Activity struct {
	BaseActivity
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
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
		// fetch body for more details
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Println("Response body:", string(bodyBytes))
		return nil, fmt.Errorf("failed to fetch activity, status code: %d", resp.StatusCode)
	}

	var activity RawActivity
	err = json.NewDecoder(resp.Body).Decode(&activity)
	if err != nil {
		return nil, err
	}

	return activity.toActivity(), nil
}

func FetchAthleteActivities(accessToken string) ([]Activity, error) {
	url := "https://www.strava.com/api/v3/athlete/activities"

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
		return nil, fmt.Errorf("failed to fetch activities, status code: %d", resp.StatusCode)
	}

	var activities []RawActivity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, err
	}

	var result []Activity
	for _, raw := range activities {
		result = append(result, *raw.toActivity())
	}

	return result, nil
}

func (r *RawActivity) toActivity() *Activity {
	startDate, err := time.Parse(time.RFC3339, r.StartDate)
	if err != nil {
		startDate = time.Time{}
	}
	endDate := startDate.Add(time.Duration(r.ElapsedTime) * time.Second)

	return &Activity{
		BaseActivity: r.BaseActivity,
		StartDate:    startDate.UTC().Format("20060102T150405Z"),
		EndDate:      endDate.UTC().Format("20060102T150405Z"),
	}
}
