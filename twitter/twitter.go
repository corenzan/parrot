package twitter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/corenzan/parrot/helpers"
	"github.com/patrickmn/go-cache"
)

var (
	ua            = "Parrot/1.0"
	endpoint      = "https://api.twitter.com"
	activityCache *cache.Cache
	logger        *log.Logger
	credentials   string
	client        *http.Client
)

// https://developer.twitter.com/en/docs/basics/authentication/overview/application-only#issuing-application-only-requests
func token() string {
	url := endpoint + "/oauth2/token"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		logger.Println("token() failed:", err)
		return ""
	}
	req.Header.Add("Authorization", "Basic "+credentials)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Add("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		logger.Println("token() failed:", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		logger.Println("token() failed:", resp.Status)
		return ""
	}
	var token map[string]string
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		logger.Println("token() failed:", err)
		return ""
	}
	if token["token_type"] != "bearer" {
		logger.Printf("token() failed: invalid token type: %v", token)
		return ""
	}
	return token["access_token"]
}

func latestActivity(username string) string {
	url := endpoint + "/1.1/statuses/user_timeline.json?count=1&screen_name=" + username
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Println("latestActivity() failed:", err)
		return ""
	}
	token := token()
	if token == "" {
		return ""
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		logger.Println("latestActivity() failed:", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		logger.Println("latestActivity() failed:", resp.Status)
		return ""
	}
	statuses := make([]map[string]interface{}, 1)
	err = json.NewDecoder(resp.Body).Decode(&statuses)
	if err != nil {
		logger.Println("latestActivity() failed:", err)
		return ""
	}
	return statuses[0]["text"].(string)
}

// ServeHTTP ...
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ext := path.Ext(r.URL.Path)
	name := strings.TrimSuffix(path.Base(r.URL.Path), ext)
	var activity string
	if value, found := activityCache.Get(name); found {
		activity = value.(string)
	} else {
		activity = latestActivity(name)
		activityCache.Set(name, activity, cache.DefaultExpiration)
	}
	if activity == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch ext {
	case ".html", "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, helpers.AutoLink(activity))
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		status := map[string]string{
			"status": activity,
		}
		err := json.NewEncoder(w).Encode(&status)
		if err != nil {
			logger.Println("ServeHTTP() failed:", err)
		}
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, activity)
	default:
		w.WriteHeader(http.StatusNotAcceptable)
	}
}

func init() {
	logger = log.New(os.Stdout, "twitter: ", log.LstdFlags)
	client = &http.Client{
		Timeout: time.Second * 2,
	}
	activityCache = cache.New(1*time.Hour, 1*time.Hour)
	key := os.Getenv("TWITTER_KEY")
	secret := os.Getenv("TWITTER_SECRET")
	credentials = base64.StdEncoding.EncodeToString([]byte(key + ":" + secret))
}
