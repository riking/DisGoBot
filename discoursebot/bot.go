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

	err = bot.RefreshCSRF()
	fatal("csrf", err)

	err = bot.Login(config)
	fatal("logging in", err)

	return bot, config
}

var bot *discourse.DiscourseSite

func main() {
	log.Info("Starting up...")
	flag.Parse()

	bot, _ := setup()

	// @BotName Match1 m a t c h 2
	// match2 extends until end of line
	mentionRegex = regexp.MustCompile(fmt.Sprintf("@%s\\s+(\\w+)\\s*((?:\\s+\\w+)*)\n", bot.Username))

//	go LikesThread(bot)
//	go GiveOutNicePosts(bot)
	bot.SubscribeNotificationPost(LikeSummon, []int{1})
	bot.SubscribeNotificationPost(OnNotifiedPost, []int{1,2,3,4,5,6,7,8,9,10,11,12})
	bot.Subscribe("/topic/1000", watchLikesThread)
	bot.Subscribe("/latest", watchLatest)
	bot.SubscribeEveryPost(OnPosted)

	bot.Start()

	time.Sleep(9 * time.Hour)
	// TODO - command line control
}

var regex = regexp.MustCompile("Since likes don't have a lot of meaning in this topic")
var mentionRegex regexp.Regexp
func OnPosted(post discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("Got post with ID", post.Id)

	if regex.MatchString(post.Raw) {
		log.Info("Found meaningless post", post.Topic_id, "/", post.Post_number, "-", "liking")
		bot.LikePost(post.Id)
	} else if (post.Topic_id == 1000) {
		log.Info("Liking likes thread post", post.Post_number)
		bot.LikePost(post.Id)
	} else if mentionRegex.MatchString(post.Raw) {
		parsed = mentionRegex.FindAllStringSubmatch(post.Raw, -1)
	}
}

func watchLikesThread(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	if msg.Data["type"] == "created" {
		id, ok := msg.Data["id"]
		if !ok {
			log.Warn("got thread message without post ID")
			return
		}
		_, ok = id.(float64)
		if !ok {
			log.Warn("got thread message without numeric post ID", id)
			return
		}
//		bot.PostHappened <- true
	}
}

func watchLatest(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	if msg.Data["message_type"] == "latest" {
		bot.PostHappened <- true
	}
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
	// TODO dead code
//	var highestSeen int = 0
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
	_ = likePosts

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
