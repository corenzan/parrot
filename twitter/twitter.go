package twitter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	// "log"
	"net/http"
	// "net/http/httputil"
	"time"
)

var (
	ua       = "Parrot/1.0"
	endpoint = "https://api.twitter.com"
)

type Client struct {
	key, secret string
	http        *http.Client
}

func New(key, secret string) *Client {
	return &Client{
		key,
		secret,
		&http.Client{
			Timeout: time.Second * 2,
		},
	}
}

type Token struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
}

type Status struct {
	Text string `json:"text"`
}

func (c *Client) Token() string {
	url := endpoint + "/oauth2/token"
	credentials := base64.StdEncoding.EncodeToString([]byte(c.key + ":" + c.secret))
	req, err := http.NewRequest("POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Basic "+credentials)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Add("User-Agent", ua)
	resp, err := c.http.Do(req)
	if err != nil {
		panic(err)
	}
	// dump, err := httputil.DumpResponse(resp, true)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf("Token: %q", dump)
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	token := &Token{}
	err = json.NewDecoder(resp.Body).Decode(token)
	if err != nil {
		panic(err)
	}
	return token.AccessToken
}

func (c *Client) LastStatus(username string) *Status {
	url := endpoint + "/1.1/statuses/user_timeline.json?count=1&screen_name=" + username
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	token := c.Token()
	if token == "" {
		return nil
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("User-Agent", ua)
	resp, err := c.http.Do(req)
	if err != nil {
		panic(err)
	}
	// dump, err := httputil.DumpResponse(resp, true)
	// if err != nil {
	// 	panic(err)
	// }
	// log.Printf("LastStatus: %q", dump)
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var statuses []*Status
	err = json.NewDecoder(resp.Body).Decode(&statuses)
	if err != nil {
		panic(err)
	}
	return statuses[0]
}
