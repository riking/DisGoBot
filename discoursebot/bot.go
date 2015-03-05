package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"
//	"regexp"
	//	"reflect"
	"os"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
//	"github.com/riking/DisGoBot/commands"
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

	bot.Subscribe("/latest", watchLatest)
	bot.Subscribe("/read-only", watchReadOnly)
	bot.SubscribeEveryPost(OnPosted)

	fatal("Starting up", bot.Start())

	select{}
	// TODO - command line control
}

const ActionTypeTopic = 4
const ActionTypePost = 5

func OnPosted(post discourse.S_Post, bot *discourse.DiscourseSite) {
    // log.Info("OnPosted got post with ID", post.Id)
	log.Debug(fmt.Sprintf("OnPosted got post {id %d topic %d num %d}", post.Id, post.Topic_id, post.Post_number))

	if (post.Post_number == 1) {
		var resp discourse.ResponseUserSerializer
		err := bot.DGetJsonTyped(fmt.Sprintf("/users/%s.json", post.Username), &resp)
		if err != nil {
			return
		}
		postCount := 0
		for _, v := range resp.User.Stats {
			if v.Action_type == ActionTypePost || v.Action_type == ActionTypeTopic {
				postCount += v.Count
			}
		}
		if postCount == 1 {
			bot.Reply(post.Topic_id, post.Post_number, "Welcome to the try.discourse.org sandbox.\n\n" +
				"Please remember that this website is a sandbox and the contents will be erased every day.")
		}
	}
}

func watchLatest(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	if msg.Data["message_type"] == "latest" {
		// log.Debug("post happened", msg)
		bot.PostHappened <- struct{}{}
	}
}

func watchReadOnly(msg discourse.S_MessageBus, bot *discourse.DiscourseSite) {
	go func() {
		time.Sleep(5 * time.Minute)
		bot.ResetPostIds <- struct{}{}
	}()
}
