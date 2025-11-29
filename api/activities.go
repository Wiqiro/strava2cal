package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BaseActivity struct {
	Id          int     `json:"id" bson:"_id"`
	Name        string  `json:"name"`
	Distance    float32 `json:"distance"`
	Elevation   float32 `json:"total_elevation_gain"`
	Timezone    string  `json:"timezone"`
	AvgSpeed    float32 `json:"average_speed"`
	AvgWatts    float32 `json:"average_watts"`
	AvgCadence  float32 `json:"average_cadence"`
	ElapsedTime int     `json:"elapsed_time"`
}

type RawActivity struct {
	BaseActivity
	Type      string `json:"sport_type"`
	StartDate string `json:"start_date"`
}

type Activity struct {
	BaseActivity `bson:",inline"`
	Type         string `json:"type"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
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
	url := "https://www.strava.com/api/v3/athlete/activities?per_page=200"

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

func formatActivityType(activityType string) string {
	typeMap := map[string]string{
		"AlpineSki":                     "Alpine Ski",
		"BackcountrySki":                "Backcountry Ski",
		"Badminton":                     "Badminton",
		"Canoeing":                      "Canoeing",
		"Crossfit":                      "Crossfit",
		"EBikeRide":                     "E-Bike Ride",
		"Elliptical":                    "Elliptical",
		"EMountainBikeRide":             "E-Mountain Bike Ride",
		"Golf":                          "Golf",
		"GravelRide":                    "Gravel Ride",
		"Handcycle":                     "Handcycle",
		"HighIntensityIntervalTraining": "High Intensity Interval Training",
		"Hike":                          "Hike",
		"IceSkate":                      "Ice Skate",
		"InlineSkate":                   "Inline Skate",
		"Kayaking":                      "Kayaking",
		"Kitesurf":                      "Kitesurf",
		"MountainBikeRide":              "Mountain Bike Ride",
		"NordicSki":                     "Nordic Ski",
		"Pickleball":                    "Pickleball",
		"Pilates":                       "Pilates",
		"Racquetball":                   "Racquetball",
		"Ride":                          "Ride",
		"RockClimbing":                  "Rock Climbing",
		"RollerSki":                     "Roller Ski",
		"Rowing":                        "Rowing",
		"Run":                           "Run",
		"Sail":                          "Sail",
		"Skateboard":                    "Skateboard",
		"Snowboard":                     "Snowboard",
		"Snowshoe":                      "Snowshoe",
		"Soccer":                        "Soccer",
		"Squash":                        "Squash",
		"StairStepper":                  "Stair Stepper",
		"StandUpPaddling":               "Stand Up Paddling",
		"Surfing":                       "Surfing",
		"Swim":                          "Swim",
		"TableTennis":                   "Table Tennis",
		"Tennis":                        "Tennis",
		"TrailRun":                      "Trail Run",
		"Velomobile":                    "Velomobile",
		"VirtualRide":                   "Virtual Ride",
		"VirtualRow":                    "Virtual Row",
		"VirtualRun":                    "Virtual Run",
		"Walk":                          "Walk",
		"WeightTraining":                "Weight Training",
		"Wheelchair":                    "Wheelchair",
		"Windsurf":                      "Windsurf",
		"Workout":                       "Workout",
		"Yoga":                          "Yoga",
	}

	if formatted, ok := typeMap[activityType]; ok {
		return formatted
	}
	return activityType
}

func (r *RawActivity) toActivity() *Activity {
	startDate, err := time.Parse(time.RFC3339, r.StartDate)
	if err != nil {
		startDate = time.Time{}
	}
	endDate := startDate.Add(time.Duration(r.ElapsedTime) * time.Second)

	activity := &Activity{
		Type:         formatActivityType(r.Type),
		BaseActivity: r.BaseActivity,
		StartDate:    startDate.UTC().Format("20060102T150405Z"),
		EndDate:      endDate.UTC().Format("20060102T150405Z"),
	}

	return activity
}
