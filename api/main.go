package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	CLIENT_ID     = os.Getenv("CLIENT_ID")
	CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
	APP_ADDRESS   = os.Getenv("APP_ADDRESS")
	MONGO_URI     = os.Getenv("MONGO_URI")
	MONGO_DB      = os.Getenv("MONGO_DB")
)

type WebhookData struct {
	AspectType     string `json:"aspect_type"`
	EventTime      int64  `json:"event_time"`
	ObjectId       int    `json:"object_id"`
	ObjectType     string `json:"object_type"`
	OwnerId        int    `json:"owner_id"`
	SubscriptionId int    `json:"subscription_id"`
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Webhook verification")
	params := r.URL.Query()
	challenge := params.Get("hub.challenge")
	if challenge != "" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"hub.challenge":"%s"}`, challenge)
		return
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	var webhookData WebhookData
	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, "Failed to parse webhook data", http.StatusBadRequest)
		return
	}

	if webhookData.ObjectType != "activity" || webhookData.AspectType != "create" {
		return
	}

	token, err := RefreshTokenIfExpired()
	if err != nil || token == nil {
		http.Error(w, "Failed to load/refresh token", http.StatusInternalServerError)
		return
	}

	activity, err := FetchActivity(token.AccessToken, webhookData.ObjectId)
	if err != nil {
		fmt.Println("Error fetching activity:", err)
		http.Error(w, "Failed to fetch activity", http.StatusInternalServerError)
		return
	}
	err = upsertActivity(activity)
	if err != nil {
		http.Error(w, "Failed to save activity", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"event added"}`))
}

func handleHook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		handlePost(w, r)
	}
	if r.Method == http.MethodGet {
		handleGet(w, r)
	}
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()
	code := params.Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}
	token, err := ExchangeCode(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	err = saveToken(token)
	if err != nil {
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func main() {
	fmt.Println("Strava To Calendar is starting...")
	fmt.Println("If you haven't already, authorize the application by visiting:")
	fmt.Printf("https://www.strava.com/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=activity:read_all\n", CLIENT_ID, APP_ADDRESS+"/auth")

	// Initialize MongoDB
	if err := initMongo(); err != nil {
		fmt.Println("Failed to init MongoDB:", err)
	} else {
		fmt.Println("MongoDB initialized")
		defer disconnectMongo()
	}

	http.Handle("/",
		http.FileServer(http.Dir("../web")),
	)
	http.HandleFunc("/auth", handleAuth)
	http.HandleFunc("/calendar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"strava.ics\"")

		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		token, err := getToken()
		if err != nil || token == nil {
			http.Error(w, "Failed to load token", http.StatusInternalServerError)
			return
		}
		activities, err := getActivities()
		if err != nil {
			http.Error(w, "Failed to load activities", http.StatusInternalServerError)
			return
		}

		icalData := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Strava To Calendar//EN\r\n"

		nowUTC := time.Now().UTC().Format("20060102T150405Z")

		for _, activity := range activities {
			var descriptionParts []string
			descriptionParts = append(descriptionParts, fmt.Sprintf("Duration: %s", (time.Duration(activity.ElapsedTime)*time.Second).String()))
			descriptionParts = append(descriptionParts, fmt.Sprintf("Distance: %.2fkm | Elevation: %.0fm", activity.Distance/1000, activity.Elevation))
			if activity.AvgSpeed > 0 {
				descriptionParts = append(descriptionParts, fmt.Sprintf("Average Speed: %.2fkm/h", activity.AvgSpeed*3.6))
			}
			if activity.AvgWatts > 0 {
				descriptionParts = append(descriptionParts, fmt.Sprintf("Average Power: %.0fW", activity.AvgWatts))
			}
			if activity.AvgCadence > 0 {
				descriptionParts = append(descriptionParts, fmt.Sprintf("Average Cadence: %.0frpm", activity.AvgCadence))
			}
			descriptionParts = append(descriptionParts, fmt.Sprintf("strava.com/activities/%d", activity.Id))
			description := escapeICalText(strings.Join(descriptionParts, "\n"))

			summary := escapeICalText(fmt.Sprintf("%s | %s", activity.Type, activity.Name))

			icalData += "BEGIN:VEVENT\r\n"
			icalData += fmt.Sprintf("UID:%d@strava2cal\r\n", activity.Id)
			icalData += fmt.Sprintf("DTSTAMP:%s\r\n", nowUTC)
			icalData += fmt.Sprintf("SUMMARY:%s\r\n", summary)
			icalData += fmt.Sprintf("DTSTART:%s\r\n", activity.StartDate)
			icalData += fmt.Sprintf("DTEND:%s\r\n", activity.EndDate)
			icalData += fmt.Sprintf("DESCRIPTION:%s\r\n", description)
			icalData += "END:VEVENT\r\n"
		}

		icalData += "END:VCALENDAR\r\n"

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(icalData))
	})
	http.HandleFunc("/hook", handleHook)
	http.HandleFunc("/auth/start", func(w http.ResponseWriter, r *http.Request) {
		redirectURL := fmt.Sprintf(
			"https://www.strava.com/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s/auth&scope=activity:read_all",
			CLIENT_ID,
			APP_ADDRESS,
		)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	http.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE")
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			err := registerWebhook(APP_ADDRESS+"/hook", "token")
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Failed to register webhook", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"webhook registered"}`))
		case http.MethodDelete:
			subId, err := getWebhook()
			if err != nil {
				http.Error(w, "Failed to load subscription id", http.StatusInternalServerError)
				return
			}
			fmt.Println("Unregistering webhook with id:", subId)
			err = unregisterWebhook(subId)
			if err != nil {
				http.Error(w, "Failed to unregister webhook", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"webhook unregistered"}`))
		}
	})
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		fmt.Println("Fetching past activities...")
		token, err := RefreshTokenIfExpired()
		if err != nil {
			http.Error(w, "Failed to refresh token", http.StatusInternalServerError)
			return
		}

		activities, err := FetchAthleteActivities(token.AccessToken)
		if err != nil {
			http.Error(w, "Failed to fetch activities", http.StatusInternalServerError)
			return
		}
		fmt.Printf("Fetched %d past activities\n", len(activities))
		if len(activities) > 0 {
			if err := setActivities(activities); err != nil {
				http.Error(w, "Failed to save activities", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"activities fetched"}`))
	})

	http.ListenAndServe(":8080", nil)

}
