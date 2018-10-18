package instagram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
	errorMissingAccessToken = errors.New("Unauthenticated")
)

// Activity ...
type Activity struct {
	Meta struct {
		ErrorType string `json:"error_type"`
	} `json:"meta"`
	Data []struct {
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
	id     string
	secret string
	cache  struct {
		activity *cache.Cache
		token    *cache.Cache
	}
	http *http.Client
	url  string
}

// New ...
func New(id, secret string) *Client {
	c := &Client{
		id:     id,
		secret: secret,
		http: &http.Client{
			Timeout: time.Second * 2,
		},
	}
	c.cache.token = cache.New(cache.NoExpiration, cache.NoExpiration)
	c.cache.activity = cache.New(1*time.Hour, 1*time.Hour)
	return c
}

// SetURL ...
func (c *Client) SetURL(r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	c.url = scheme + "://" + r.Host
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

// RedirectURL ...
func (c *Client) RedirectURL(ext string) string {
	return c.url + "/instagram/?ext=" + ext
}

// AuthenticationEndpoint ...
func (c *Client) AuthenticationEndpoint(ext string) string {
	u := url.QueryEscape(c.RedirectURL(ext))
	return endpoint + "/oauth/authorize/?client_id=" + c.id + "&redirect_uri=" + u + "&response_type=code"
}

// AccessTokenResponse ...
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	User        struct {
		Name string `json:"username"`
	} `json:"user"`
}

// SaveAccessToken ...
func (c *Client) SaveAccessToken(code, ext string) (string, error) {
	payload := url.Values{}
	payload.Set("client_id", c.id)
	payload.Set("client_secret", c.secret)
	payload.Set("grant_type", "authorization_code")
	payload.Set("redirect_uri", c.RedirectURL(ext))
	payload.Set("code", code)
	req, err := c.NewRequest("POST", "/oauth/access_token", strings.NewReader(payload.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		dump, _ := httputil.DumpResponse(resp, true)
		return "", errors.New(string(dump))
	}
	atr := &AccessTokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(atr)
	if err != nil {
		return "", err
	}
	c.cache.token.Set(atr.User.Name, atr.AccessToken, cache.DefaultExpiration)
	return atr.User.Name, nil
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
		req, err := c.NewRequest("GET", "/v1/users/self/media/recent/?count=5&access_token="+token, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			dump, _ := httputil.DumpResponse(resp, true)
			return nil, errors.New(string(dump))
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
	id := os.Getenv("INSTAGRAM_ID")
	secret := os.Getenv("INSTAGRAM_SECRET")
	client = New(id, secret)
}

// ServeHTTP ...
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("error") == "access_denied" {
		http.Error(w, r.FormValue("error_description"), http.StatusUnauthorized)
		return
	}
	client.SetURL(r)
	basename := path.Base(r.URL.Path)
	ext := path.Ext(basename)
	if code := r.FormValue("code"); code != "" {
		ext = r.FormValue("ext")
		username, err := client.SaveAccessToken(code, ext)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Location", client.url+"/instagram/"+username+ext)
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}
	username := strings.TrimSuffix(basename, ext)
	if username == "" {
		http.NotFound(w, r)
		return
	}
	activity, err := client.LatestActivity(username)
	if err == errorMissingAccessToken {
		w.Header().Set("Location", client.AuthenticationEndpoint(ext))
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	} else if err != nil {
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
