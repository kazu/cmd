package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	twitterscraper "github.com/n0madic/twitter-scraper"
)

func main() {

	fmt.Printf("params = %s\n", strings.Join(os.Args[1:], " "))

	for tweet := range twitterscraper.SearchTweets(context.Background(),
		fmt.Sprintf("%s  -filter:retweets", strings.Join(os.Args[1:], " ")), 50) {
		if tweet.Error != nil {
			panic(tweet.Error)
		}
		a := tweet.HTML
		fmt.Println("------------\n")
		fmt.Println(a)
	}
}
