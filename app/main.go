package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

type Item struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Text   string `json:"text"`
	Link   string `json:"link"`
	UserID int    `json:"userid"`
}

type Feed struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Url    string `json:"url"`
	UserID int    `json:"userid"`
}

type ItemResponse struct {
	Item Item `json:"item"`
	Feed Feed `json:"feed"`
}

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/**/*")

	router.GET("/view/item", handleRequest)

	router.Static("/static", "./static")

	fmt.Println("http://localhost:8080/?id=263443")
	router.Run(":8080")
}

func handleRequest(c *gin.Context) {
	resp := getItem(c.Query(("id")))

	if resp.Feed.Type == "tiktok" {
		c.Redirect(http.StatusFound, resp.Item.Link)
		return
	}

	if resp.Feed.Type == "kinogo" {
		imgSrcs, _ := getImageSrcs(resp.Item.Link)
		posterSrc := "https://kinogo.fm" + imgSrcs[0]
		c.HTML(http.StatusOK, "kinogo", gin.H{
			"feed_title": resp.Feed.Title,
			"poster_src": posterSrc,
			"title":      resp.Item.Title,
			"message":    resp.Item.Text,
			"link":       resp.Item.Link,
		})
		return
	}

	if resp.Feed.Type == "yt" {
		posterSrc := getYouTubeThumbnailURL(resp.Item.Link)
		c.HTML(http.StatusOK, "yt", gin.H{
			"feed_title": resp.Feed.Title,
			"poster_src": posterSrc,
			"title":      resp.Item.Title,
			"message":    resp.Item.Text,
			"link":       resp.Item.Link,
		})

	}

	if resp.Feed.Type == "vk" {
		imgSrcs, _ := getImageSrcs(resp.Item.Link)
		// TODO: dont use magick number
		posterSrc := imgSrcs[3]
		c.HTML(http.StatusOK, "vk", gin.H{
			"feed_title": resp.Feed.Title,
			"poster_src": posterSrc,
			"title":      resp.Item.Title,
			"message":    resp.Item.Text,
			"link":       resp.Item.Link,
		})
	}

	if resp.Feed.Type == "stories" {
		storyType, _ := checkContentType(resp.Item.Link)
		fmt.Println(storyType)
		c.HTML(http.StatusOK, "stories", gin.H{
			"feed_title":   resp.Feed.Title,
			"poster_src":   resp.Item.Link,
			"content_type": storyType,
			"title":        resp.Item.Title,
			"message":      resp.Item.Text,
			"link":         resp.Item.Link,
		})
	}

	c.Redirect(http.StatusFound, resp.Item.Link)
}

func getItem(id string) ItemResponse {
	url := "http://feed.gws.freemyip.com/api/v1/item?id=" + id
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		panic("request failed")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		panic("request failed")
	}

	var jsonData ItemResponse
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Println("Error:", err)
		panic("request failed")
	}

	return jsonData
}

func getImageSrcs(url string) ([]string, error) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.79 Safari/537.36"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
	}
	defer resp.Body.Close()

	htmlContent, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	var srcs []string

	var findImages func(*html.Node)
	findImages = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					srcs = append(srcs, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findImages(c)
		}
	}

	findImages(htmlContent)

	return srcs, nil
}

func getYouTubeVideoID(videoURL string) string {
	parts := strings.Split(videoURL, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "watch?v=") {
			return strings.TrimPrefix(part, "watch?v=")
		}
	}
	return ""
}

func getYouTubeThumbnailURL(videoURL string) string {
	videoID := getYouTubeVideoID(videoURL)
	return fmt.Sprintf("https://i3.ytimg.com/vi/%s/maxresdefault.jpg", videoID)
}

func checkContentType(url string) (string, error) {
	// Send an HTTP HEAD request to get the content type
	resp, err := http.Head(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Extract the content type from the response header
	contentType := resp.Header.Get("Content-Type")

	// Check if the content type indicates an image
	if strings.HasPrefix(contentType, "image/") {
		return "img", nil
	}

	// Check if the content type indicates a video
	if strings.HasPrefix(contentType, "video/") {
		return "video", nil
	}

	// If neither image nor video, return unknown
	return "Unknown", nil
}
