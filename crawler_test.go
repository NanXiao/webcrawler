package webcrawler

import (
	"io"
	"strings"
	"testing"
)

// FakeFetcher is a function that implements an interface using the same strategy of http.HandlerFunc.
type FakeFetcher func(url string) (io.Reader, error)

func (f FakeFetcher) Fetch(url string) (io.Reader, error) {
	return f(url)
}

func TestCrawler(t *testing.T) {
	testData := []struct {
		url      string
		data     string
		expected Page
	}{
		{
			url: "example.com",
			data: `<html>
  <head>
    <link rel="stylesheet" type="text/css" href="example.css">
  </head>
  <body>
    <a href="example.com/test">Example</a>
    <img src="example.png" alt="example"/>
    <script type="text/javascript" src="example.js"/>
  </body>
</html>`,
			expected: Page{
				URL: "example.com",
				Links: []Link{
					{
						Page: &Page{
							URL: "example.com/test",
							StaticAssets: []string{
								"example.css",
								"example.png",
								"example.js",
							},
						},
					},
				},
				StaticAssets: []string{
					"example.css",
					"example.png",
					"example.js",
				},
			},
		},
	}

	for _, testItem := range testData {
		page := Crawl(testItem.url, FakeFetcher(func(url string) (io.Reader, error) {
			return strings.NewReader(testItem.data), nil
		}))

		if page.Fail {
			t.Fatalf("Unexpected error returned")
		}

		if page.String() != testItem.expected.String() {
			t.Errorf("Unexpected page returned. Expected '%s' and got '%s'",
				testItem.expected, page)
		}
	}
}
