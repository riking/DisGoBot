package commands

import (
	"strconv"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

func init() {
	CommandMap["likeme"] = likeme
	CommandMap["likethat"] = likethat
	CommandMap["likepost"] = likepost
}


func likeme(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	bot.LikePost(post.Id)
	log.Info("liked post", post.Id, "by likeme command")
}


func likethat(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	repliedPost, err := bot.GetPostByNumber(post.Topic_id, post.Reply_to_post_number)
	if err == nil {
		bot.LikePost(repliedPost.Id)
		log.Info("liked post", repliedPost.Id, "by likethat command")
	}
}


func likepost(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
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
