package utils

import (
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type HTMLMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
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

func ExtractHtmlMetaString(htmlString string) HTMLMeta {
	return ExtractHtmlMeta(strings.NewReader(htmlString))
}

func ExtractHtmlMetaURL(url string) (HTMLMeta, error) {
	res, err := http.Get(url)
	if err != nil {
		return HTMLMeta{}, err
	}

	defer res.Body.Close()
	return ExtractHtmlMeta(res.Body), nil
}
