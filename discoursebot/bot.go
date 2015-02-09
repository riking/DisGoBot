package main

import (
	"flag"
	"io/ioutil"
	"os"

	"encoding/json"

	"github.com/riking/discourse/discourse"
	"fmt"
	"time"
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

var bot discourse.DiscourseSite

func main() {
	fmt.Println("Starting up...")
	flag.Parse()

	bot, _ := setup()

	var highestSeen int = 0
	likePosts := func (post discourse.S_Post) {
		var err error
		if post.Like_count >= 7 && post.Like_count < 10 {
			fmt.Println("Post id", post.Id, "has", post.Like_count, "likes - liking")
			err = bot.LikePost(post.Id)
			if err != nil {
				panic(err)
			}
		}
	}
	discourse.SeeEveryPost(bot, &highestSeen, likePosts, 224382);
	discourse.SeeEveryPost(bot, &highestSeen, likePosts, 0);

	time.Sleep(0)
}


