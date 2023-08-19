package utils

import "testing"

func TestTitle(t *testing.T) {
	title := "test"

	data := ExtractHtmlMetaString(`<html><head><meta property="og:title" content="` + title + `"></head></html>`)
	if title != data.Title {
		t.Errorf("Title is not correct")
	}

	data = ExtractHtmlMetaString("<html><head><title>" + title + "</title></head></html>")
	if title != data.Title {
		t.Errorf("Title is not correct")
	}
}

func TestDescription(t *testing.T) {
	description := "description"

	data := ExtractHtmlMetaString(`<html><head><meta property="og:description" content="` + description + `"></head></html>`)
	if description != data.Description {
		t.Errorf("Description is not correct")
	}

	data = ExtractHtmlMetaString(`<html><head><meta property="description" content="` + description + `"></head></html>`)
	if description != data.Description {
		t.Errorf("Description is not correct")
	}
}

func TestImage(t *testing.T) {
	url := "/test.png"

	data := ExtractHtmlMetaString(`<html><head><meta property="og:image" content="` + url + `"></head></html>`)
	if url != data.Image {
		t.Errorf("Image is not correct")
	}
}
