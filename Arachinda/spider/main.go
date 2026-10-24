package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

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
	page, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP: %w,", err)
	}
	defer page.Body.Close()
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

func findImgs(node *html.Node, images *[]string) {
	// check if img for current node
	if node.Type == html.ElementNode && node.Data == "img" {
		for _, attr := range node.Attr {
			if attr.Key == "src" && attr.Val != "" {
				*images = append(*images, attr.Val)
				break
			}
		}
	}
	// visit every child node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		findImgs(child, images)	
	}
}

func extractImages(pageURL string, htmlContent string) ([]string, error) {
	doc, err := parseHTML(htmlContent)
	if err != nil {
		return nil, err
	}
	var images []string		
	findImgs(doc, &images)

	return images, nil
}


func main() {
	// parse flags and url
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", cfg)

	// get the html of page
	html, err := getPage(cfg.url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(html)
	// get the images if found
	images, err := extractImages(cfg.url, html)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(images)
}
