package main

import (
	"flag"
	"io/ioutil"
	"os"

	"encoding/json"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
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
		log.Error(desc, err)
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
	log.Info("Starting up...")
	flag.Parse()

	bot, _ := setup()

//	go LikesThread(bot)
//	go GiveOutNicePosts(bot)
	bot.SubscribeNotificationPost(LikeSummon, []int{1})
	bot.SubscribeNotificationPost(OnNotifiedPost, []int{1,2,3,4,5,6,7,8,9,10,11,12})

	discourse.OnNotification <- true
	time.Sleep(9 * time.Hour)
	// TODO - command line control
}

func OnNotifiedPost(notification discourse.S_Notification, post discourse.S_Post, bot *discourse.DiscourseSite) () {
	log.Info("Got notification of type", discourse.NotificationTypesInverse[notification.Notification_type])
	log.Info("Post is id", post.Id)
	// TODO do something ?
}

func LikeSummon(notification discourse.S_Notification, post discourse.S_Post, bot *discourse.DiscourseSite) {
	fmt.Println("LikeSummon got notification")
	if post.Reply_to_post_number > 0 {
		fmt.Println("liking post it is reply to")

		var postToLike discourse.S_Post
		err := bot.DGetJsonTyped(fmt.Sprintf("/posts/by_number/%d/%d", post.Topic_id, post.Reply_to_post_number), &postToLike)
		if err != nil {
			log.Error("LikeSummon - failed to load post", err)
			return
		}
		err = bot.LikePost(postToLike.Id)
		if err != nil {
			log.Error("LikeSummon - liking post", err)
			return
		}
	}
}



func GiveOutNicePosts(bot *discourse.DiscourseSite) {
	var highestSeen int = 0
	regex := regexp.MustCompile("(?i)purple")

	likePosts := func(post discourse.S_Post, bot *discourse.DiscourseSite) {
		var err error
		if post.Like_count == 9 {
			err = bot.LikePost(post.Id)
			fmt.Println("[INFO]", "Liked post id", post.Id, "which had", post.Like_count, "likes")

			if _, ok := err.(discourse.ErrorRateLimit); ok {
				fmt.Println("[WARN]", "Reached rate limit, sleeping 1 hour")
				time.Sleep(1 * time.Hour)
			} else if err != nil {
				panic(err)
			}
		}
		if regex.MatchString(post.Raw) {
			err = bot.LikePost(post.Id)
			fmt.Println("[INFO]", "Liked purple post with id", post.Id)
			if _, ok := err.(discourse.ErrorRateLimit); ok {
				fmt.Println("[WARN]", "Reached rate limit, sleeping 1 hour")
				time.Sleep(1 * time.Hour)
			} else if err != nil {
				panic(err)
			}
		}
	}

	// highestSeen = SOME_VALUE // load from persistent store?
	// TODO change change change!
	func() {
		discourse.SeeEveryPost(bot, &highestSeen, likePosts, 188682);
		for {
			discourse.SeeEveryPost(bot, &highestSeen, likePosts, 0);
			time.Sleep(15 * time.Minute)
		}
	}()
}

func LikesThread(bot *discourse.DiscourseSite) {
	return
	var response discourse.ResponseTopic
	err := bot.DGetJsonTyped("/t/1000.json", &response)
	if err != nil {
		fmt.Println("[ERR]", err)
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
		fmt.Println("[INFO]", "Liked post id", postId, "in Likes thread (#", idx, ")")
		time.Sleep(200 * time.Millisecond)

		if _, ok := err.(discourse.ErrorRateLimit); ok {
			fmt.Println("[WARN]", "Reached rate limit, sleeping 1 hour")
			time.Sleep(1 * time.Hour)
		} else if err != nil {
			panic(err)
		}
	}
}
