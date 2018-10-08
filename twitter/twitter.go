package twitter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	ua       = "Parrot/1.0"
	endpoint = "https://api.twitter.com"
)

// Activity ...
type Activity struct {
	Status string `json:"status"`
}

// HTML ...
func (a *Activity) HTML() string {
	return a.Status
}

// Text ...
func (a *Activity) Text() string {
	return a.Status
}

// Client ...
type Client struct {
	credentials string
	cache       *cache.Cache
	http        *http.Client
}

// New ...
func New(key, secret string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: time.Second * 2,
		},
		cache:       cache.New(1*time.Hour, 1*time.Hour),
		credentials: base64.StdEncoding.EncodeToString([]byte(key + ":" + secret)),
	}
}

// NewRequest ...
func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(endpoint + path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", ua)
	return req, nil
}

// AccessToken ...
func (c *Client) AccessToken() (string, error) {
	req, err := c.NewRequest("POST", "/oauth2/token", bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Basic "+c.credentials)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	var token map[string]string
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return "", err
	}
	if token["token_type"] != "bearer" {
		return "", errors.New("Token type is not bearer")
	}
	return token["access_token"], nil
}

// Twit ...
type Twit struct {
	Text string `json:"text"`
}

// Timeline ...
type Timeline []Twit

// LatestActivity ...
func (c *Client) LatestActivity(username string) (*Activity, error) {
	var activity *Activity
	if value, ok := c.cache.Get(username); ok {
		activity = value.(*Activity)
	} else {
		req, err := c.NewRequest("GET", "/1.1/statuses/user_timeline.json?count=1&screen_name="+username, nil)
		if err != nil {
			return nil, err
		}
		token, err := c.AccessToken()
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(resp.Status)
		}
		timeline := Timeline{}
		err = json.NewDecoder(resp.Body).Decode(&timeline)
		if err != nil {
			return nil, err
		}
		activity = &Activity{
			Status: timeline[0].Text,
		}
		c.cache.Set(username, activity, cache.DefaultExpiration)
	}
	return activity, nil
}

var (
	client *Client
)

func init() {
	secret := os.Getenv("TWITTER_SECRET")
	key := os.Getenv("TWITTER_KEY")

	client = New(key, secret)
}

// ServeHTTP ...
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	basename := path.Base(r.URL.Path)
	ext := path.Ext(basename)
	username := strings.TrimSuffix(basename, ext)
	if username == "" {
		http.NotFound(w, r)
		return
	}

	activity, err := client.LatestActivity(username)
	if err != nil {
		panic(err)
	}

	switch ext {
	case ".html", "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, activity.HTML())
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err := json.NewEncoder(w).Encode(activity)
		if err != nil {
			panic(err)
		}
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, activity.Text())
	default:
		w.WriteHeader(http.StatusNotAcceptable)
	}
}
