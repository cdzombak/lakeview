package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Description  string         `xml:"description"`
	PubDate      string         `xml:"pubDate"`
	Link         string         `xml:"link"`
	MediaContent []MediaContent `xml:"http://search.yahoo.com/mrss/ content"`
}

type MediaContent struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Medium string `xml:"medium,attr"`
}

type Photo struct {
	URL     string
	PubDate string
	Link    string
}

func main() {
	feeds := []string{
		"https://mastodon.social/@livelakehuron.rss",
		"https://mastodon.social/@livelakemichigan.rss",
		"https://mastodon.social/@livelakesuperior.rss",
		"https://mastodon.social/@livelakeerie.rss",
		"https://mastodon.social/@livelakeontario.rss",
	}

	var allPhotos []Photo
	for _, feedURL := range feeds {
		photos, err := fetchPhotos(feedURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", feedURL, err)
			continue
		}
		allPhotos = append(allPhotos, photos...)
	}

	if len(allPhotos) == 0 {
		fmt.Fprintf(os.Stderr, "No photos found\n")
		os.Exit(1)
	}

	sort.Slice(allPhotos, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC1123Z, allPhotos[i].PubDate)
		tj, _ := time.Parse(time.RFC1123Z, allPhotos[j].PubDate)
		return ti.After(tj)
	})

	if err := generateHTML(allPhotos, "index.html"); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated index.html successfully with %d photos\n", len(allPhotos))
}

func fetchPhotos(url string) ([]Photo, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS: %w", err)
	}

	var photos []Photo

	for _, item := range rss.Channel.Items {
		for _, media := range item.MediaContent {
			if media.Medium == "image" {
				photos = append(photos, Photo{
					URL:     media.URL,
					PubDate: item.PubDate,
					Link:    item.Link,
				})
			}
		}
	}

	return photos, nil
}

func generateHTML(photos []Photo, outputFile string) error {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Great Lakes Live Photos</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: #f5f5f5;
            padding: 20px;
        }

        h1 {
            text-align: center;
            margin-bottom: 30px;
            color: #333;
        }

        .masonry {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 15px;
            grid-auto-flow: dense;
        }

        @media (max-width: 1200px) {
            .masonry {
                grid-template-columns: repeat(3, 1fr);
            }
        }

        @media (max-width: 768px) {
            .masonry {
                grid-template-columns: repeat(2, 1fr);
            }
        }

        @media (max-width: 480px) {
            .masonry {
                grid-template-columns: 1fr;
            }
        }

        .photo-item {
            background: white;
            border-radius: 8px;
            overflow: hidden;
            will-change: box-shadow;
        }

        .photo-item:hover {
            box-shadow: 0 2px 12px rgba(0,0,0,0.2);
        }

        .photo-item img {
            width: 100%;
            display: block;
        }

        .photo-item a {
            display: block;
        }
    </style>
</head>
<body>
    <h1>Great Lakes Live Photos</h1>
    <div class="masonry">
        {{range .}}
        <div class="photo-item">
            <a href="{{.Link}}" target="_blank" rel="noopener noreferrer">
                <img src="{{.URL}}" alt="Photo from {{.PubDate}}" loading="lazy">
            </a>
        </div>
        {{end}}
    </div>
</body>
</html>
`

	t, err := template.New("page").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if err := t.Execute(f, photos); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
