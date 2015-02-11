package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"
	"regexp"
	//	"reflect"
	"os"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
	"github.com/riking/DisGoBot/commands"
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
	mentionRegex = regexp.MustCompile(fmt.Sprintf("(?i)!%s\\s+(\\S+)([^\n]*)", bot.Username))

	bot.Subscribe("/topic/1000", watchLikesThread)
	bot.Subscribe("/latest", watchLatest)
	bot.SubscribeEveryPost(OnPosted)
	bot.SubscribeEveryPost(CheckForCommand)

	fatal("Starting up", bot.Start())

	time.Sleep(9 * time.Hour)
	// TODO - command line control
}

var regex = regexp.MustCompile("Since likes don't have a lot of meaning in this topic")
var mentionRegex *regexp.Regexp
func OnPosted(post discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("OnPosted got post with ID", post.Id)

	if regex.MatchString(post.Raw) {
		log.Info("Found meaningless post", post.Topic_id, "/", post.Post_number, "-", "liking")
		bot.LikePost(post.Id)
	} else if (post.Topic_id == 1000) {
		actions := post.Actions_summary
		for _, act := range actions {
			if act.Id == 2 {
				if act.Can_act {
					log.Info("Liking likes thread post", post.Post_number)
					bot.LikePost(post.Id)
				} else {
					log.Debug("Found already-liked likes thread post", post.Post_number)
				}
				break
			}
		}
	} else {
	}
}

func CheckForCommand(post discourse.S_Post, bot *discourse.DiscourseSite) {
	if mentionRegex.MatchString(post.Raw) {
		parsed := mentionRegex.FindAllStringSubmatch(post.Raw, 10)

		go commands.RunCommandBatch(parsed, post, bot)
	} else {
		log.Debug("no command found")
	}

}

func watchLikesThread(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	if msg.Data["type"] == "created" {
		id, ok := msg.Data["id"]
		if !ok {
			log.Warn("got thread message without post ID")
			return
		}
		idNum, ok := id.(float64)
		if !ok {
			log.Warn("got thread message without numeric post ID", id)
			return
		}
		log.Info("Liking likes thread post", idNum)
		bot.LikePost(int(idNum))
	}
}

func watchLatest(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	if msg.Data["message_type"] == "latest" {
		log.Debug("post happened", msg)
		bot.PostHappened <- true
	}
}
