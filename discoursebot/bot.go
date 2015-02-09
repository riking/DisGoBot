package main

import (
	"flag"
	"io/ioutil"
	"os"

	"encoding/json"

	"github.com/riking/discourse/discourse"
	"fmt"
	"time"
	"regexp"
	//	"reflect"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.json", "configuration file to load")
}

func fatal(desc string, err error) {
	if err != nil {
		fmt.Println("Fatal: ", desc, err)
		panic(err)
	}
}

func setup() (bot *discourse.DiscourseSite, config discourse.Config) {
	file, err := os.Open(configFile)
	fatal("open config", err)
	jsonBlob, err := ioutil.ReadAll(file)
	fatal("read config", err)

	err = json.Unmarshal(jsonBlob, &config)
	fatal("parse config", err)

	bot, err = discourse.NewDiscourseSite(config)
	fatal("setting up bot", err)

	err = bot.Login(config)
	fatal("logging in", err)

	return bot, config
}

var bot *discourse.DiscourseSite

func main() {
	fmt.Println("Starting up...")
	flag.Parse()

	bot, _ := setup()

	go LikesThread(bot)
	go GiveOutNicePosts(bot)


	time.Sleep(9 * time.Hour)
}

func GiveOutNicePosts(bot *discourse.DiscourseSite) {
	var highestSeen int = 0
	regex := regexp.MustCompile("(?i)purple")

	likePosts := func(post discourse.S_Post) {
		var err error
		if post.Like_count >= 9 && post.Like_count < 10 {
			err = bot.LikePost(post.Id)
			fmt.Println("Liked post id", post.Id, "which had", post.Like_count, "likes")

			if _, ok := err.(discourse.ErrorRateLimit); ok {
				fmt.Println("Reached rate limit, sleeping 1 hour")
				time.Sleep(1 * time.Hour)
			} else if err != nil {
				panic(err)
			}
		}
		if regex.MatchString(post.Raw) {
			err = bot.LikePost(post.Id)
			fmt.Println("Liked purple post with id", post.Id)
			if _, ok := err.(discourse.ErrorRateLimit); ok {
				fmt.Println("Reached rate limit, sleeping 1 hour")
				time.Sleep(1 * time.Hour)
			} else if err != nil {
				panic(err)
			}
		}
	}
	func() {
		discourse.SeeEveryPost(bot, &highestSeen, likePosts, 192732);
		discourse.SeeEveryPost(bot, &highestSeen, likePosts, 0);
	}()
}

func LikesThread(bot *discourse.DiscourseSite) {
	return
	var response discourse.ResponseTopic
	err := bot.DGetJsonTyped("/t/1000.json", &response)
	if err != nil {
		fmt.Println(err)
		return
	}
	var highestLikedPost int = 12900
	var highestLikedPostNumber int = 100
	for idx, postId := range response.Post_stream.Stream {
		if idx < highestLikedPostNumber {
			continue
		}
		if postId < highestLikedPost {
			continue
		}
		highestLikedPostNumber = idx
		highestLikedPost = postId
		err = bot.LikePost(postId)
		fmt.Println("Liked post id", postId, "in Likes thread (#", idx, ")")
		time.Sleep(200 * time.Millisecond)

		if _, ok := err.(discourse.ErrorRateLimit); ok {
			fmt.Println("Reached rate limit, sleeping 1 hour")
			time.Sleep(1 * time.Hour)
		} else if err != nil {
			panic(err)
		}
	}
}
