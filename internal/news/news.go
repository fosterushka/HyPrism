package news

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

const CDN_URL = "https://cdn.hytale.com/variants/blog_thumb_"

type coverImage struct {
	S3Key string `json:"s3Key"`
}

type NewsItem struct {
	Title       string     `json:"title"`
	BodyExcerpt string     `json:"bodyExcerpt"`
	Excerpt     string     `json:"excerpt"`
	URL         string     `json:"url"`
	Date        string     `json:"date"`
	PublishedAt string     `json:"publishedAt"`
	Slug        string     `json:"slug"`
	CoverImage  coverImage `json:"coverImage"`
	Author      string     `json:"author"`
	ImageURL    string     `json:"imageUrl"`
}

// FetchNews fetches news from hytale.com blog api
func FetchNews(limit int) ([]NewsItem, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://hytale.com/api/blog/post/published?limit=%d", limit), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "HyPrism/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	items, err := parseNewsJSON(string(body))
	if err != nil {
		return nil, err
	}

	return items, nil
}

func parseNewsJSON(body string) ([]NewsItem, error) {
	var items []NewsItem
	json.Unmarshal([]byte(body), &items)

	for idx := range items {
		var err error
		parsedUrl, err := parseUrl(items[idx].PublishedAt, items[idx].Slug)
		if err != nil {
			return nil, err
		}
		parsedDate, err := parseDate(items[idx].PublishedAt)
		if err != nil {
			return nil, err
		}
		items[idx].Date = parsedDate
		items[idx].URL = parsedUrl
		items[idx].Excerpt = html.UnescapeString(items[idx].BodyExcerpt)
		items[idx].ImageURL = CDN_URL + items[idx].CoverImage.S3Key

	}

	return items, nil
}
func parseUrl(publishedDate, slug string) (string, error) {
	parsedDate, err := time.Parse(time.RFC3339, publishedDate)
	if err != nil {
		return "", err
	}
	//https://hytale.com/news/2026/1/hytale-patch-notes-update-1
	return fmt.Sprintf("https://hytale.com/news/%d/%d/%s", parsedDate.Year(), parsedDate.Month(), slug), nil
}

// it follows the same format as hytale blog post does
func parseDate(publishedDate string) (string, error) {
	parsedDate, err := time.Parse(time.RFC3339, publishedDate)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s %d", parsedDate.Month().String(), addOrdinal(parsedDate.Day()), parsedDate.Year()), nil
}

// takes a number and adds its ordinal (like st or th) to the end.
func addOrdinal(n int) string {
	switch n {
	case 1, 21, 31:
		return fmt.Sprintf("%dst", n)
	case 2, 22:
		return fmt.Sprintf("%dnd", n)
	case 3, 23:
		return fmt.Sprintf("%drd", n)
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// NewsService provides news fetching capabilities
type NewsService struct {
	cache     []NewsItem
	cacheTime time.Time
	cacheTTL  time.Duration
}

// NewNewsService creates a new news service
func NewNewsService() *NewsService {
	return &NewsService{
		cacheTTL: 5 * time.Minute,
	}
}

// GetNews returns cached news or fetches new news if cache is expired
func (s *NewsService) GetNews(limit int) ([]NewsItem, error) {
	// Check cache
	if time.Since(s.cacheTime) < s.cacheTTL && len(s.cache) > 0 {
		if len(s.cache) > limit {
			return s.cache[:limit], nil
		}
		return s.cache, nil
	}

	// Fetch fresh news
	items, err := FetchNews(limit)
	if err != nil {
		// Return cached data if available
		if len(s.cache) > 0 {
			return s.cache, nil
		}
		return nil, err
	}

	// Update cache
	s.cache = items
	s.cacheTime = time.Now()

	return items, nil
}
