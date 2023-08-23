package utils

import (
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"

type HTMLMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	ThemeColor  string `json:"themeColor"`
}

func ExtractHtmlMeta(resp io.Reader) (data HTMLMeta) {
	tokenizer, titleFound := html.NewTokenizer(resp), false
	for {
		switch tokenizer.Next() {
		case html.StartTagToken, html.SelfClosingTagToken:
			t := tokenizer.Token()

			switch t.Data {
			case "body":
				return
			case "title":
				titleFound = true
			case "meta":
				attrs := map[string]string{}

				for _, attr := range t.Attr {
					attrs[attr.Key] = attr.Val
				}

				if property, ok := attrs["property"]; ok {
					if content, ok := attrs["content"]; ok {
						switch property {
						case "description":
							data.Description = content
						case "og:title":
							data.Title = content
						case "og:description":
							data.Description = content
						case "og:image":
							data.Image = content
						case "theme-color":
							data.ThemeColor = content
						}
					}
				}
			}
		case html.TextToken:
			if titleFound {
				data.Title = tokenizer.Token().Data
				titleFound = false
			}
		case html.ErrorToken:
			return
		}
	}
}

func ExtractHtmlMetaFromString(htmlString string) HTMLMeta {
	return ExtractHtmlMeta(strings.NewReader(htmlString))
}

func ExtractHtmlMetaFromURL(url string) (HTMLMeta, error) {
	client := http.Client{Timeout: time.Second * 10}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return HTMLMeta{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return HTMLMeta{}, err
	}

	defer res.Body.Close()
	return ExtractHtmlMeta(res.Body), nil
}
