package webcrawler

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	// Limit the number of goroutines to avoid running out of File descriptors.
	maxOutstanding = 256
)

var (
	// Semaphore to control goroutine execution.
	sem = make(chan int, maxOutstanding)
)

func init() {
	for i := 0; i < maxOutstanding; i++ {
		sem <- 1
	}
}

// Fetcher creates an interface to allow a flexibility on how we retrieve the page data. For tests
// we will simulate the response while in production we will do a HTTP GET.
type Fetcher interface {
	Fetch(url string) (io.Reader, error)
}

// HTTPFetcher will retrieve the page content via HTTP GET request.
type HTTPFetcher struct {
}

func (f HTTPFetcher) Fetch(url string) (io.Reader, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}

// Link stores information of other URLs in this page.
type Link struct {
	Page       *Page // Page information about the other URL.
	CyclicPage bool  // Flag to indicate if this page has already been processed.
}

type Page struct {
	URL          string   // Address of the page.
	Fail         bool     // Flag to indicate that the system failed to access the URL.
	Links        []Link   // List of links for other URLs in this page.
	StaticAssets []string // List of static dependencies of this page.
}

// String method transforms the Page into text mode to print the results.
func (p Page) String() string {
	staticAssets := ""
	for _, staticAsset := range p.StaticAssets {
		if len(staticAssets) > 0 {
			staticAssets += "\n"
		}

		staticAssets += fmt.Sprintf("  StaticAsset:  %s", staticAsset)
	}

	links := ""
	for _, link := range p.Links {
		if len(links) > 0 {
			links += "\n"
		}

		linkPage := ""

		// Check for nil pointer because there can be links without href (anchors).
		if link.Page != nil {
			if link.CyclicPage {
				// Don't print already visited pages to avoid infinite recursion.
				linkPage = fmt.Sprintf("\n    Page: %s", link.Page.URL)

			} else {
				// Add an identification level to the link content.
				linkPage = strings.Replace(link.Page.String(), "\n", "\n    ", -1)
			}
		}

		links += fmt.Sprintf("  Links in current page: %s", linkPage)
	}

	pageStr := ""
	if p.Fail {
		pageStr = fmt.Sprintf("\nPage: %s (Failed to get this URL)\n", p.URL)
	} else {
		pageStr = fmt.Sprintf("\nPage: %s\n", p.URL)
	}

	// Don't add unnecessary spaces when there's no information
	if len(staticAssets) > 0 {
		pageStr += "\n" + staticAssets + "\n"
	}

	// Don't add unnecessary spaces when there's no information
	if len(links) > 0 {
		pageStr += "\n" + links + "\n"
	}

	return pageStr
}

type crawler struct {
	domain  string
	fetcher Fetcher
	wg      sync.WaitGroup

	// visitedPages store all pages already visited in a map, so that if we found a link for the same
	// page again, we just pick on the map the same object address. The function that prints the page
	// is responsible for detecting cycle loops.
	visitedPages map[string]*Page

	// visitedPagesLock allows visitedPages to be manipulated safely by different goroutines.
	visitedPagesLock sync.Mutex
}

func Crawl(url string, fetcher Fetcher) *Page {
	c := &crawler{domain: url, fetcher: fetcher, visitedPages: make(map[string]*Page)}

	c.wg.Add(1)
	p := &Page{URL: url}
	c.visitedPages[url] = p
	go crawlPage(c, p)
	c.wg.Wait()

	return p
}

func crawlPage(c *crawler, page *Page) {
	<-sem

	defer func() {
		sem <- 1
		c.wg.Done()
	}()

	r, err := c.fetcher.Fetch(c.domain)
	if err != nil {
		page.Fail = true
		return
	}

	root, err := html.Parse(r)
	if err != nil {
		page.Fail = true
		return
	}

	parseHTML(c, root, page)
}

// parseHTML is an auxiliary function of Crawl function that will travel recursively
// around the HTML document identifying elements to populate the Page object.
func parseHTML(c *crawler, node *html.Node, page *Page) {
	if node.Type == html.ElementNode {
		switch node.Data {
		case "a":
			var link Link
			for _, attr := range node.Attr {
				if attr.Key != "href" {
					continue
				}

				linkURL := strings.TrimSpace(attr.Val)
				if strings.HasPrefix(linkURL, "/") {
					linkURL = c.domain + linkURL
				}

				if strings.HasPrefix(linkURL, c.domain) {
					ok := true
					c.visitedPagesLock.Lock()

					if _, ok = c.visitedPages[linkURL]; ok {
						link.Page = page
						link.CyclicPage = true
					} else {
						link.Page = &Page{
							URL: linkURL,
						}

						c.visitedPages[linkURL] = link.Page
					}
					c.visitedPagesLock.Unlock()

					if !ok {
						page.Links = append(page.Links, link)
						c.wg.Add(1)
						go crawlPage(c, link.Page)
					}
				}
				break
			}

		case "link":
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					page.StaticAssets = append(page.StaticAssets, attr.Val)
				}
			}

		case "img", "script":
			for _, attr := range node.Attr {
				if attr.Key == "src" {
					page.StaticAssets = append(page.StaticAssets, attr.Val)
				}
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		parseHTML(c, child, page)
	}
}
