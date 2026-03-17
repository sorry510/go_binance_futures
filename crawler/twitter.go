package crawler

import (
	"encoding/xml"
	"fmt"
	"go_binance_futures/models"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

var (
	statusIDReg = regexp.MustCompile(`/status/(\d+)`)
	htmlTagReg  = regexp.MustCompile(`<[^>]*>`)
)

func StartTwitterCrawler() {
	if !getCrawlerEnable() {
		logs.Info("twitter crawler disabled")
		return
	}

	accounts := getTwitterAccounts()
	if len(accounts) == 0 {
		logs.Warn("twitter crawler accounts is empty, skip")
		return
	}

	interval := getCrawlerInterval()
	logs.Info("twitter crawler start, interval:", interval.String(), "accounts:", strings.Join(accounts, ","))

	run := func() {
		for _, account := range accounts {
			inserted, err := crawlAccount(account)
			if err != nil {
				logs.Error("twitter crawler account error:", account, err)
				continue
			}
			if inserted > 0 {
				logs.Info("twitter crawler inserted:", inserted, "account:", account)
			}
		}
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		run()
	}
}

func crawlAccount(account string) (int, error) {
	baseURLs := getTwitterRSSBaseURLs()
	var body []byte
	var err error

	for _, baseURL := range baseURLs {
		url := fmt.Sprintf("%s/%s/rss", strings.TrimRight(baseURL, "/"), account)
		body, err = requestRSS(url)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}

	var feed rssFeed
	if err = xml.Unmarshal(body, &feed); err != nil {
		return 0, err
	}

	inserted := 0
	for _, item := range feed.Channel.Items {
		if saveNews(account, item) {
			inserted++
		}
	}
	return inserted, nil
}

func saveNews(account string, item rssItem) bool {
	o := orm.NewOrm()
	externalID := extractStatusID(item.Link)
	now := time.Now().UnixMilli()
	publishedAt := parsePubDate(item.PubDate)
	if publishedAt == 0 {
		publishedAt = now
	}

	query := o.QueryTable(new(models.News)).Filter("type", "twitter").Filter("source", account)
	if externalID != "" {
		query = query.Filter("external_id", externalID)
	} else {
		query = query.Filter("url", item.Link)
	}

	exists := models.News{}
	err := query.One(&exists, "id")
	if err == nil {
		return false
	}
	if err != orm.ErrNoRows {
		logs.Error("twitter crawler query news error:", err)
		return false
	}

	content := strings.TrimSpace(item.Description)
	content = htmlTagReg.ReplaceAllString(content, "")

	news := models.News{
		Type:        "twitter",
		Source:      account,
		ExternalID:  externalID,
		Title:       strings.TrimSpace(item.Title),
		Content:     content,
		Url:         strings.TrimSpace(item.Link),
		PublishedAt: publishedAt,
		CreateTime:  now,
		UpdateTime:  now,
	}
	if _, err = o.Insert(&news); err != nil {
		logs.Error("twitter crawler insert news error:", err)
		return false
	}
	return true
}

func extractStatusID(link string) string {
	m := statusIDReg.FindStringSubmatch(link)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func parsePubDate(pubDate string) int64 {
	pubDate = strings.TrimSpace(pubDate)
	if pubDate == "" {
		return 0
	}

	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822}
	for _, layout := range layouts {
		t, err := time.Parse(layout, pubDate)
		if err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

func getCrawlerEnable() bool {
	enable, err := config.Int("crawler::twitter_enable")
	if err != nil {
		return false
	}
	return enable == 1
}

func getCrawlerInterval() time.Duration {
	intervalSec, err := config.String("crawler::twitter_interval_sec")
	if err != nil {
		return 60 * time.Second
	}
	n, err := strconv.Atoi(strings.TrimSpace(intervalSec))
	if err != nil || n <= 0 {
		return 60 * time.Second
	}
	return time.Duration(n) * time.Second
}

func getTwitterAccounts() []string {
	val, err := config.String("crawler::twitter_accounts")
	if err != nil {
		return []string{"binance", "coinbase", "ethereum", "solana"}
	}
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.TrimPrefix(p, "@"))
		if p == "" {
			continue
		}
		res = append(res, p)
	}
	if len(res) == 0 {
		return []string{"binance", "coinbase", "ethereum", "solana"}
	}
	return res
}

func getTwitterRSSBaseURLs() []string {
	val, err := config.String("crawler::twitter_rss_base_urls")
	if err != nil {
		return []string{"https://nitter.net", "https://nitter.poast.org"}
	}
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		res = append(res, p)
	}
	if len(res) == 0 {
		return []string{"https://nitter.net", "https://nitter.poast.org"}
	}
	return res
}

func requestRSS(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; go_binance_futures/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rss request status=%d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
