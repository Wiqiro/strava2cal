package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var (
	CLIENT_ID     = os.Getenv("CLIENT_ID")
	CLIENT_SECRET = os.Getenv("CLIENT_SECRET")
	APP_ADDRESS   = os.Getenv("APP_ADDRESS")
	CALENDAR_FILE = os.Getenv("CALENDAR_FILE")
	STATE_FILE    = os.Getenv("STATE_FILE")
)

type WebhookData struct {
	AspectType     string `json:"aspect_type"`
	EventTime      int64  `json:"event_time"`
	ObjectId       int    `json:"object_id"`
	ObjectType     string `json:"object_type"`
	OwnerId        int    `json:"owner_id"`
	SubscriptionId int    `json:"subscription_id"`
}

func addEvent(f *os.File, activity *Activity) error {
	_, err := f.Seek(-int64(len("END:VCALENDAR\n")), io.SeekEnd)
	if err != nil {
		return err
	}

	event := fmt.Sprintf("BEGIN:VEVENT\nSUMMARY:%s\nDTSTART:%s\nDTEND:%s\nEND:VEVENT\nEND:VCALENDAR\n",
		activity.Name,
		activity.StartDate,
		activity.StartDate,
	)

	_, err = f.WriteString(event)
	if err != nil {
		return err
	}
	return nil
}

func registerWebhook(clientId, clientSecret, callbackUrl, verifyToken string) error {
	webhookUrl := "https://www.strava.com/api/v3/push_subscriptions"

	req, err := http.NewRequest("POST", webhookUrl, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
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

	SaveSubscriptionID(content.Id)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register webhook, status code: %d", resp.StatusCode)
	}
	return nil
}

func unregisterWebhook(clientId, clientSecret string, subscriptionId int) error {
	webhookUrl := fmt.Sprintf("https://www.strava.com/api/v3/push_subscriptions/%d", subscriptionId)

	req, err := http.NewRequest("DELETE", webhookUrl, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("client_id", clientId)
	q.Add("client_secret", clientSecret)
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

func handlePost(w http.ResponseWriter, r *http.Request, icsFile string) {
	file, err := os.OpenFile(icsFile, os.O_RDWR, 0644)
	if err != nil {
		http.Error(w, "Failed to open calendar file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var webhookData WebhookData
	err = json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, "Failed to parse webhook data", http.StatusBadRequest)
		return
	}

	if webhookData.ObjectType != "activity" || webhookData.AspectType != "create" {
		return
	}

	token, err := LoadToken()
	if err != nil {
		http.Error(w, "Failed to load token", http.StatusInternalServerError)
		return
	}
	activity, err := FetchActivity(token.AccessToken, webhookData.ObjectId)
	if err != nil {
		fmt.Println("Error fetching activity:", err)
		http.Error(w, "Failed to fetch activity", http.StatusInternalServerError)
		return
	}

	err = addEvent(file, activity)
	if err != nil {
		http.Error(w, "Failed to add event", http.StatusInternalServerError)
		return
	}

	fmt.Println("Event added for activity:", activity.Name)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"event added"}`))
}

func handleHook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		handlePost(w, r, CALENDAR_FILE)
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
	token, err := ExchangeCode(CLIENT_ID, CLIENT_SECRET, code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	err = SaveToken(token)
	if err != nil {
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func main() {
	fmt.Println("Strava To Calendar is starting...")
	fmt.Println("If you haven't already, authorize the application by visiting:")
	fmt.Printf("https://www.strava.com/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=activity:read_all\n", CLIENT_ID, APP_ADDRESS+"/auth")

	if _, err := os.Stat(CALENDAR_FILE); os.IsNotExist(err) {
		file, err := os.Create(CALENDAR_FILE)
		if err != nil {
			panic(err)
		}
		file.WriteString("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//strava-to-calendar//EN\nEND:VCALENDAR\n")
		file.Close()
	}

	http.HandleFunc("/hook", handleHook)
	http.HandleFunc("/auth", handleAuth)
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
			err := registerWebhook(CLIENT_ID, CLIENT_SECRET, APP_ADDRESS+"/hook", "token")
			if err != nil {
				http.Error(w, "Failed to register webhook", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"webhook registered"}`))
		case http.MethodDelete:
			id := LoadSubscriptionID()
			if id == nil {
				http.Error(w, "No subscription ID found", http.StatusBadRequest)
				return
			}
			err := unregisterWebhook(CLIENT_ID, CLIENT_SECRET, *id)
			if err != nil {
				http.Error(w, "Failed to unregister webhook", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"webhook unregistered"}`))
		}
	})
	http.ListenAndServe(":8080", nil)

}
