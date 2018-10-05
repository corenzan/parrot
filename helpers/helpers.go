package helpers

import (
	"net/url"
	"path"

	"mvdan.cc/xurls"
)

var (
	pat = xurls.Relaxed()
)

// AutoLink replaces all URLs found in text with links and return the new string.
func AutoLink(text string) string {
	return pat.ReplaceAllStringFunc(text, func(src string) string {
		u, err := url.Parse(src)
		if err != nil {
			return src
		}
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		return `<a href="` + u.String() + `">` + src + `</a>`
	})
}

// AutoImg replaces all images URLs found in text with img tags and return the new string.
func AutoImg(text string) string {
	return pat.ReplaceAllStringFunc(text, func(src string) string {
		u, err := url.Parse(src)
		if err != nil {
			return src
		}
		switch path.Ext(u.Path) {
		case ".gif", ".png", ".jpg", ".jpeg":

		}
		if u.Scheme == "" {
			u.Scheme = "http"
		}
		return `<a href="` + u.String() + `">` + src + `</a>`
	})
}
