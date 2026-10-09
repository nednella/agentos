package news

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type feedItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

// parseFeed returns the issues the feed lists, in its order, with no items yet.
func parseFeed(body []byte) ([]Issue, error) {
	var feed struct {
		Items []feedItem `xml:"channel>item"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("reading the feed: %w", err)
	}
	var issues []Issue
	for _, it := range feed.Items {
		link := strings.TrimSpace(it.Link)
		date := link[strings.LastIndex(link, "/")+1:]
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			continue
		}
		issues = append(issues, Issue{Date: date, Title: strings.TrimSpace(it.Title), URL: link})
	}
	if len(issues) == 0 {
		return nil, errors.New("the feed lists no issues")
	}
	return issues, nil
}

var readTime = regexp.MustCompile(`^(.*?)\s*\((\d+ minute read)\)$`)

// parsePage returns the stories of an issue page, without the sponsors.
func parsePage(body []byte, date string) ([]Item, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("reading the page: %w", err)
	}
	var items []Item
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			if it, ok := parseArticle(n); ok {
				it.ID = fmt.Sprintf("%s-%d", date, len(items))
				items = append(items, it)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(items) == 0 {
		return nil, errors.New("the page has no stories")
	}
	return items, nil
}

func parseArticle(article *html.Node) (Item, bool) {
	link := find(article, func(n *html.Node) bool { return n.Data == "a" && attr(n, "href") != "" })
	heading := find(article, func(n *html.Node) bool { return n.Data == "h3" })
	if link == nil || heading == nil {
		return Item{}, false
	}
	title := text(heading)
	if strings.Contains(title, "(Sponsor)") {
		return Item{}, false
	}
	href, ok := cleanURL(attr(link, "href"))
	if !ok {
		return Item{}, false
	}
	it := Item{Title: title, URL: href}
	if m := readTime.FindStringSubmatch(title); m != nil {
		it.Title, it.ReadTime = m[1], m[2]
	}
	if body := find(article, func(n *html.Node) bool { return strings.Contains(" "+attr(n, "class")+" ", " newsletter-html ") }); body != nil {
		it.Summary = text(body)
	}
	return it, true
}

// cleanURL drops the newsletter's tracking parameter and refuses what is not a web link.
func cleanURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	var kept []string
	for _, pair := range strings.Split(u.RawQuery, "&") {
		if pair != "" && pair != "utm_source=tldrdev" && pair != "utm_source=tldrnewsletter" {
			kept = append(kept, pair)
		}
	}
	u.RawQuery = strings.Join(kept, "&")
	return u.String(), true
}

func find(n *html.Node, match func(*html.Node) bool) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && match(c) {
			return c
		}
		if f := find(c, match); f != nil {
			return f
		}
	}
	return nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// text is the text under n with the white space collapsed.
func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
