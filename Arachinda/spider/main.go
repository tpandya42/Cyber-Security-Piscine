package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"path"
	"path/filepath"

	"golang.org/x/net/html"
)

type conf struct {
	url string
	depth	int
	recursion bool
	path 	string
}

func parseArgs(args []string) (conf, error) {
	if len(args) < 1 {
		return conf{}, fmt.Errorf("No URL provided")
	}

	cfg := conf {
		depth: 5,
		recursion: false,
		path: "./data/",
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-r":
			cfg.recursion = true

		case "-l":
			if i+1 >= len(args) {
				return conf{}, fmt.Errorf("missing value for -l")
			}
			depth, err := strconv.Atoi(args[i + 1])
			if err != nil {
				return conf{}, fmt.Errorf("invalid depth :%s", args[i+1])
			}
			if depth < 0 {
				return conf{}, fmt.Errorf("invalid depth")	
			}
			cfg.depth = depth
			i++

		case "-p":
			if i+1 >= len(args) {
				return conf{}, fmt.Errorf("missing value for -p")
			}
			cfg.path = args[i + 1]
			i++

		default:
			if strings.HasPrefix(args[i], "-") {
				return conf{}, fmt.Errorf("unknown option: %s", args[i])
			}
			if cfg.url != "" {
				return conf{}, fmt.Errorf("multiple URLs provided")
			}
			//cfg.url = args[i]
			parsedURL, err := url.ParseRequestURI(args[i])
			if err != nil {
				return conf{}, fmt.Errorf("invalid url")
			}
			if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
				return conf{}, fmt.Errorf("URL must include http:// or https:// and a hostname")
			}
			cfg.url = args[i]	
		}
	}
	if cfg.url == "" {
		return conf{}, fmt.Errorf("no URL provided")
	}
	return cfg, nil
}

func getPage(url string) (string, error) {
	// send an http get request to the url 
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w,", err)
	}
	req.Header.Set("User-Agent", "Spider")
	client := &http.Client {
		Timeout: 15 * time.Second,
	}

	page, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending requenst: %w", err)
	}
	// check the status code
	if page.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected status code: %d (%s)", page.StatusCode, page.Status)
	}
	// read the body
	res, err := io.ReadAll(page.Body)
	if err != nil {
		return "", fmt.Errorf("Read failed: %w,", err)
	}
	return string(res), nil
	
}

func parseHTML(content string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func isImageExt(ur string) (bool) {
	parsedURL, err := url.Parse(ur)
	if err != nil {
		return false
	}
	ext := strings.ToLower(path.Ext(parsedURL.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func downloadImage(imgURL string, savePath string) error {
	if err := os.MkdirAll(savePath, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, imgURL, nil)
	if err != nil {
		return fmt.Errorf("create image get req: %w", err)
	}
	req.Header.Set("User-Agent", "Spider")
	
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read image body: %w", err)
	}

	parsedURL, err := url.Parse(imgURL)
	if err != nil {
		return fmt.Errorf("parse image URL: %w", err)
	}

	filename := path.Base(parsedURL.Path)
	if filename == "." || filename == "/" || filename == "" {
		return fmt.Errorf("could not determine image filename from URL")
	}

	filePath := filepath.Join(savePath, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write image file: %w", err)
	}

	return nil
}

func findImgs(node *html.Node,baseURL *url.URL, images *[]string) {
	// check if img for current node
	if node.Type == html.ElementNode && node.Data == "img" {
		for _, attr := range node.Attr {
			if attr.Key == "src" && attr.Val != "" {
				imgURL, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				absoluteURL := baseURL.ResolveReference(imgURL)
				if isImageExt(absoluteURL.String()) {
					*images = append(*images, absoluteURL.String())
				}
				break
			}
		}
	}
	// visit every child node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		findImgs(child,baseURL, images)	
	}
}

func extractImages(pageURL string, htmlContent string) ([]string, error) {
	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid page url: %w", err)
	}
	doc, err := parseHTML(htmlContent)
	if err != nil {
		return nil, err
	}
	var images []string		
	findImgs(doc, baseURL, &images)

	return images, nil
}

func findLinks(node *html.Node, baseURL *url.URL, links *[]string) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" && attr.Val != "" {
				linkURL, err := url.Parse(attr.Val)
				if err != nil {
					continue
				}
				absoluteURL := baseURL.ResolveReference(linkURL)
				*links = append(*links, absoluteURL.String())
				break
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		findLinks(child, baseURL, links)
	}
}

func extractLinks(pageURL string, htmlContent string) ([]string, error) {
	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid page url: %w", err)
	}
	doc, err := parseHTML(htmlContent)
	if err != nil {
		return nil, err
	}
	var links []string
	findLinks(doc, baseURL, &links)
	return links, nil
}

func sameHost(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return ua.Host == ub.Host
}

func scrapper(pageURL string, depth int, visited map[string]bool, seedURL string, cfg conf) {
	if depth > cfg.depth {
		return
	}
	if visited[pageURL] {
		return
	}
	visited[pageURL] = true

	html, err := getPage(pageURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch %s: %v\n", pageURL, err)
		return
	}

	images, err := extractImages(pageURL, html)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract images from %s: %v\n", pageURL, err)
		return
	}

	for _, imgURL := range images {
		if err := downloadImage(imgURL, cfg.path); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to download %s: %v\n", imgURL, err)
			continue
		}
		fmt.Printf("Downloaded: %s\n", imgURL)
	}

	if !cfg.recursion {
		return
	}

	links, err := extractLinks(pageURL, html)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to extract links from %s: %v\n", pageURL, err)
		return
	}

	for _, link := range links {
		if !sameHost(seedURL, link) {
			continue
		}
		scrapper(link, depth+1, visited, seedURL, cfg)
	}
}

func main() {
	// parse flags and url
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", cfg)

	visited := make(map[string]bool)
	scrapper(cfg.url, 0, visited, cfg.url, cfg)
}
