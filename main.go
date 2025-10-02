package main

import (
	"encoding/xml"
	"flag"
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
	outputFile := flag.String("out", "index.html", "Output HTML file path")
	flag.Parse()

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

	if err := generateHTML(allPhotos, *outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s successfully with %d photos\n", *outputFile, len(allPhotos))
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
    <meta http-equiv="refresh" content="1800">
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

        .masonry {
            position: relative;
        }

        .photo-item {
            position: absolute;
            width: calc(25% - 12px);
            background: white;
            border-radius: 8px;
            overflow: hidden;
            will-change: box-shadow;
        }

        @media (max-width: 1200px) {
            .photo-item {
                width: calc(33.333% - 10px);
            }
        }

        @media (max-width: 768px) {
            .photo-item {
                width: calc(50% - 8px);
            }
        }

        @media (max-width: 480px) {
            .photo-item {
                width: 100%;
            }
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
    <div class="masonry">
        {{range .}}
        <div class="photo-item">
            <a href="{{.Link}}" target="_blank" rel="noopener noreferrer">
                <img src="{{.URL}}" alt="Photo from {{.PubDate}}" loading="lazy">
            </a>
        </div>
        {{end}}
    </div>
    <script>
        function layoutMasonry() {
            const container = document.querySelector('.masonry');
            const items = Array.from(document.querySelectorAll('.photo-item'));
            const gap = 15;

            let columnCount = 4;
            if (window.innerWidth <= 480) columnCount = 1;
            else if (window.innerWidth <= 768) columnCount = 2;
            else if (window.innerWidth <= 1200) columnCount = 3;

            const columnHeights = new Array(columnCount).fill(0);
            const columnPositions = [];
            for (let i = 0; i < columnCount; i++) {
                columnPositions.push(i * (100 / columnCount));
            }

            items.forEach((item, index) => {
                if (index < items.length) {
                    const img = item.querySelector('img');
                    if (img.complete) {
                        positionItem(item, img);
                    } else {
                        img.addEventListener('load', () => positionItem(item, img));
                    }
                }
            });

            function positionItem(item, img) {
                const minColumnIndex = columnHeights.indexOf(Math.min(...columnHeights));
                const itemHeight = img.naturalHeight * (item.offsetWidth / img.naturalWidth);

                item.style.left = columnPositions[minColumnIndex] + '%';
                item.style.top = columnHeights[minColumnIndex] + 'px';

                columnHeights[minColumnIndex] += itemHeight + gap;
            }

            setTimeout(() => {
                const maxHeight = Math.max(...columnHeights);
                container.style.height = maxHeight + 'px';
            }, 100);
        }

        window.addEventListener('load', layoutMasonry);
        window.addEventListener('resize', layoutMasonry);
    </script>
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
