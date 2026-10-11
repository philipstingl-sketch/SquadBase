package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GameNews struct {
	Game    string `json:"game"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Date    string `json:"date"`
	Source  string `json:"source"`
	Summary string `json:"summary"`
}
type NewsData struct {
	Items     []GameNews `json:"items"`
	CheckedAt string     `json:"checkedAt"`
	Error     string     `json:"error"`
}
type NewsEngine struct {
	mu     sync.Mutex
	data   NewsData
	next   time.Time
	client *http.Client
}

var newsTags = regexp.MustCompile(`<[^>]*>`)

func cleanNews(s string) string {
	s = html.UnescapeString(newsTags.ReplaceAllString(s, " "))
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > 200 {
		s = string(r[:200]) + "…"
	}
	return s
}
func newNews() *NewsEngine { return &NewsEngine{client: &http.Client{Timeout: 12 * time.Second}} }
func (n *NewsEngine) get(force bool) NewsData {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !force && time.Now().Before(n.next) {
		return n.data
	}
	// Public Steam announcements; no fabricated patch/version claims.
	sources := []struct {
		id    string
		appID int
	}{{"gta", 271590}, {"rocket", 252950}}
	type result struct {
		items []GameNews
		err   error
	}
	ch := make(chan result, len(sources)+1)
	for _, src := range sources {
		go func(game string, id int) { items, e := n.fetchSteam(game, id); ch <- result{items, e} }(src.id, src.appID)
	}
	go func() { items, e := n.fetchMinecraft(); ch <- result{items, e} }()
	items := []GameNews{}
	errs := []string{}
	for i := 0; i < len(sources)+1; i++ {
		r := <-ch
		items = append(items, r.items...)
		if r.err != nil {
			errs = append(errs, r.err.Error())
		}
	}
	// Latest-first for all games, based on dates returned by Steam.
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Date > items[i].Date {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if len(items) > 0 {
		n.data.Items = items
		n.data.CheckedAt = time.Now().Format(time.RFC3339)
	}
	n.data.Error = strings.Join(errs, " · ")
	if len(items) > 0 {
		n.next = time.Now().Add(20 * time.Minute)
	} else {
		n.next = time.Now().Add(2 * time.Minute)
	}
	return n.data
}
func (n *NewsEngine) fetchSteam(game string, id int) ([]GameNews, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	defer cancel()
	u := "https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?" + url.Values{"appid": {fmt.Sprint(id)}, "count": {"7"}, "maxlength": {"450"}}.Encode()
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return nil, e
	}
	res, e := n.client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("Steam-News %s: %w", game, e)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Steam-News %s HTTP %d", game, res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 250000))
	if e != nil {
		return nil, e
	}
	var payload struct {
		AppNews struct {
			NewsItems []struct {
				Title     string `json:"title"`
				URL       string `json:"url"`
				Date      int64  `json:"date"`
				FeedLabel string `json:"feedlabel"`
				Contents  string `json:"contents"`
			} `json:"newsitems"`
		} `json:"appnews"`
	}
	if e = json.Unmarshal(b, &payload); e != nil {
		return nil, e
	}
	out := []GameNews{}
	for _, it := range payload.AppNews.NewsItems {
		// Don't use URLs returned by feed unless they use HTTPS, to avoid unsafe links.
		addr, e := url.Parse(it.URL)
		if e != nil || addr.Scheme != "https" || addr.Host == "" {
			continue
		}
		if it.Title == "" || it.Date <= 0 {
			continue
		}
		out = append(out, GameNews{Game: game, Title: html.UnescapeString(it.Title), URL: it.URL, Date: time.Unix(it.Date, 0).UTC().Format(time.RFC3339), Source: "Steam · " + it.FeedLabel, Summary: cleanNews(it.Contents)})
	}
	return out, nil
}
func (n *NewsEngine) handleAPI(w http.ResponseWriter, r *http.Request, name string) {
	if n == nil {
		failure(w, 503, "Updates noch nicht bereit")
		return
	}
	if name == "news" && r.Method == "GET" {
		send(w, 200, n.get(false))
		return
	}
	if name == "news/refresh" && r.Method == "POST" {
		send(w, 200, n.get(true))
		return
	}
	failure(w, 405, "Unbekannter News-Aufruf")
}

// Minecraft provides an official publisher RSS feed. Read the real published
// dates and article links; never synthesize a release version or patch status.
func (n *NewsEngine) fetchMinecraft() ([]GameNews, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "GET", "https://www.minecraft.net/en-us/feeds/community-content/rss", nil)
	if e != nil {
		return nil, e
	}
	res, e := n.client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("Minecraft-RSS: %w", e)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Minecraft-RSS HTTP %d", res.StatusCode)
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 512*1024))
	if e != nil {
		return nil, e
	}
	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				PubDate     string `xml:"pubDate"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if e = xml.Unmarshal(data, &rss); e != nil {
		return nil, fmt.Errorf("Minecraft-RSS ungültig: %w", e)
	}
	items := []GameNews{}
	for _, it := range rss.Channel.Items {
		if len(items) >= 10 {
			break
		}
		u, e := url.Parse(strings.TrimSpace(it.Link))
		if e != nil || u.Scheme != "https" || (u.Host != "www.minecraft.net" && u.Host != "minecraft.net") {
			continue
		}
		t, e := time.Parse(time.RFC1123Z, strings.TrimSpace(it.PubDate))
		if e != nil {
			t, e = time.Parse(time.RFC1123, strings.TrimSpace(it.PubDate))
		}
		if e != nil {
			t, e = time.Parse(time.RFC3339, strings.TrimSpace(it.PubDate))
		}
		if e != nil {
			continue
		}
		if it.Title == "" {
			continue
		}
		items = append(items, GameNews{Game: "minecraft", Title: html.UnescapeString(it.Title), URL: u.String(), Date: t.UTC().Format(time.RFC3339), Summary: cleanNews(it.Description), Source: "Minecraft.net · RSS"})
	}
	return items, nil
}
