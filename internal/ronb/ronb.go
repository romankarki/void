package ronb

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Article struct {
	Title string
	URL   string
	Image string
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func FetchNews(page int) ([]Article, error) {
	url := "https://www.ronbpost.com/category/news/"
	if page > 1 {
		url = fmt.Sprintf("https://www.ronbpost.com/category/news/page/%d/", page)
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return parseArticles(string(body)), nil
}

func FetchArticleContent(url string) (string, string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}

	html := string(body)
	image := extractArticleImage(html)
	return parseArticleContent(html), image, nil
}

func extractArticleImage(html string) string {
	imgRegex := regexp.MustCompile(`<img[^>]+class="[^"]*wp-image[^"]*"[^>]+src="([^"]+)"`)
	matches := imgRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	fallbackRegex := regexp.MustCompile(`<img[^>]+src="([^"]+)"[^>]*class="[^"]*`)
	matches = fallbackRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func parseArticles(html string) []Article {
	var articles []Article

	articleRegex := regexp.MustCompile(`<article[^>]*class="[^"]*post[^"]*"[^>]*>[\s\S]*?<a href="([^"]+)"[^>]*>[\s\S]*?<img[^>]+src="([^"]+)"[^>]*>[\s\S]*?<h3[^>]*><a[^>]*>([^<]+)</a></h3>`)
	matches := articleRegex.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) >= 4 {
			url := m[1]
			image := m[2]
			title := strings.TrimSpace(m[3])
			title = cleanHTML(title)
			if title != "" && !seen[title] {
				seen[title] = true
				articles = append(articles, Article{Title: title, URL: url, Image: image})
			}
		}
	}

	if len(articles) == 0 {
		titleRegex := regexp.MustCompile(`<h3><a href="([^"]+)"[^>]*>([^<]+)</a></h3>`)
		titleMatches := titleRegex.FindAllStringSubmatch(html, -1)
		for _, m := range titleMatches {
			if len(m) >= 3 {
				url := m[1]
				title := strings.TrimSpace(m[2])
				title = cleanHTML(title)
				if title != "" && !seen[title] {
					seen[title] = true
					articles = append(articles, Article{Title: title, URL: url})
				}
			}
		}
	}

	return articles
}

func parseArticleContent(html string) string {
	contentRegex := regexp.MustCompile(`<div class="post-entry">([\s\S]*?)</div>`)
	matches := contentRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		content := matches[1]
		content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, "")
		content = strings.ReplaceAll(content, "&nbsp;", " ")
		content = strings.ReplaceAll(content, "&hellip;", "...")
		content = strings.ReplaceAll(content, "&#039;", "'")
		content = strings.ReplaceAll(content, "&quot;", "\"")
		content = strings.ReplaceAll(content, "\n\n", "\n")
		content = strings.TrimSpace(content)
		return content
	}

	fallbackRegex := regexp.MustCompile(`<p>([^<]+)</p>`)
	allMatches := fallbackRegex.FindAllStringSubmatch(html, -1)
	var paragraphs []string
	for _, m := range allMatches {
		if len(m) > 1 && len(m[1]) > 50 {
			text := m[1]
			text = strings.ReplaceAll(text, "&nbsp;", " ")
			text = strings.ReplaceAll(text, "&#039;", "'")
			paragraphs = append(paragraphs, text)
		}
	}
	if len(paragraphs) > 0 {
		return strings.Join(paragraphs, "\n\n")
	}

	return "Could not extract article content."
}

func cleanHTML(s string) string {
	s = strings.ReplaceAll(s, "&#039;", "'")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return s
}
