package commands // import "github.com/riking/DisGoBot/commands"

import (
	"strings"
	"strconv"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

func HasCommand(commandName string) bool {
	commandName = strings.ToLower(commandName)
	if commandName == "seen" {
		return true
	}
	if commandName == "likeme" {
		return true
	}
	if commandName == "likethat" {
		return true
	}
	if commandName == "likepost" {
		return true
	}
	return false
}

func RunCommand(commandName string, extraArgs string, post discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("Processing command", commandName, "with args", extraArgs)
	commandName = strings.ToLower(commandName)
	splitArgs := strings.Split(extraArgs, " ")

	if commandName == "likeme" {
		bot.LikePost(post.Id)
		log.Info("liked post", post.Id, "by likeme command")
	} else if commandName == "likethat" {
		repliedPost, err := bot.GetPostByNumber(post.Topic_id, post.Reply_to_post_number)
		if err == nil {
			bot.LikePost(repliedPost.Id)
			log.Info("liked post", repliedPost.Id, "by likethat command")
		}
	} else if commandName == "likepost" {
		if len(splitArgs) < 3 {
			return
		}
		topicId, err := strconv.Atoi(splitArgs[1])
		if err != nil {
			return
		}
		postNum, err := strconv.Atoi(splitArgs[2])
		if err != nil {
			return
		}
		postToLike, err := bot.GetPostByNumber(topicId, postNum)
		if err != nil {
			return
		}
		bot.LikePost(postToLike.Id)
		log.Info("liked post", postToLike.Id, "by likepost command")
	}
}
