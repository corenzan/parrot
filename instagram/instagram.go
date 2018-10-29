package instagram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	cache "github.com/patrickmn/go-cache"
)

const (
	ua       = "Parrot/1.0"
	endpoint = "https://api.instagram.com"
)

var (
	errorMissingAccessToken = errors.New("Missing Access Token")
	errorEmpty              = errors.New("Empty")
)

// Activity ...
type Activity struct {
	Meta struct {
		ErrorType string `json:"error_type"`
	} `json:"meta"`
	Data []struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Link   string `json:"link"`
		Images struct {
			StandardResolution struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"standard_resolution"`
		} `json:"images"`
		Type string `json:"type"`
	} `json:"data"`
}

// String ...
func (a *Activity) String() string {
	o := []string{}
	for _, data := range a.Data {
		o = append(o, data.Images.StandardResolution.URL)
	}
	return strings.Join(o, "\n")
}

// HTML ...
func (a *Activity) HTML() string {
	o := ""
	for _, data := range a.Data {
		l := data.Link
		s := data.Images.StandardResolution.URL
		w := data.Images.StandardResolution.Width
		h := data.Images.StandardResolution.Height
		o += fmt.Sprintf(`<a href="%s"><img src="%s" width="%d" height="%d"></a>`, l, s, w, h)
	}
	return o
}

// JSON ...
func (a *Activity) JSON() []map[string]string {
	o := []map[string]string{}
	for _, data := range a.Data {
		o = append(o, map[string]string{
			"href":   data.Link,
			"src":    data.Images.StandardResolution.URL,
			"width":  fmt.Sprint(data.Images.StandardResolution.Width),
			"height": fmt.Sprint(data.Images.StandardResolution.Height),
		})
	}
	return o
}

// Client ...
type Client struct {
	cache struct {
		activity *cache.Cache
		token    *cache.Cache
	}
	http *http.Client
}

// New ...
func New() *Client {
	c := &Client{
		http: &http.Client{
			Timeout: time.Second * 2,
		},
	}
	c.cache.token = cache.New(cache.NoExpiration, cache.NoExpiration)
	c.cache.activity = cache.New(1*time.Hour, 1*time.Hour)
	return c
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

// SaveAccessToken ...
func (c *Client) SaveAccessToken(token string) (string, error) {
	req, err := c.NewRequest("GET", "/v1/users/self/media/recent/?count=9&access_token="+token, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	var activity *Activity
	err = json.NewDecoder(resp.Body).Decode(&activity)
	if err != nil {
		return "", err
	}
	if len(activity.Data) == 0 {
		return "", errorEmpty
	}
	username := activity.Data[0].User.Username
	c.cache.token.Set(username, token, cache.DefaultExpiration)
	c.cache.activity.Set(username, activity, cache.DefaultExpiration)
	return username, nil
}

// LatestActivity ...
func (c *Client) LatestActivity(username string) (*Activity, error) {
	var activity *Activity
	if value, ok := c.cache.activity.Get(username); ok {
		activity = value.(*Activity)
	} else {
		var token string
		if value, ok := c.cache.token.Get(username); ok {
			token = value.(string)
		} else {
			return nil, errorMissingAccessToken
		}
		req, err := c.NewRequest("GET", "/v1/users/self/media/recent/?count=9&access_token="+token, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(resp.Status)
		}
		err = json.NewDecoder(resp.Body).Decode(&activity)
		if err != nil {
			return nil, err
		}
		c.cache.activity.Set(username, activity, cache.DefaultExpiration)
	}
	return activity, nil
}

var (
	client *Client
)

func init() {
	client = New()
}

// ServeHTTP ...
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	basename := path.Base(r.URL.Path)
	if r.Method == http.MethodPost && basename == "instagram" {
		username, err := client.SaveAccessToken(r.FormValue("token"))
		if err != nil {
			panic(err)
		}
		w.Header().Set("Location", "/instagram/"+username)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	ext := path.Ext(basename)
	username := strings.TrimSuffix(basename, ext)
	if username == "instagram" {
		http.NotFound(w, r)
		return
	}
	activity, err := client.LatestActivity(username)
	if err != nil {
		if err == errorMissingAccessToken {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
		panic(err)
	}
	switch ext {
	case ".html", "":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, activity.HTML())
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err := json.NewEncoder(w).Encode(activity.JSON())
		if err != nil {
			panic(err)
		}
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, activity.String())
	default:
		w.WriteHeader(http.StatusNotAcceptable)
	}
}
