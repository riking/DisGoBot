package commands

import (
	"strconv"

	// "github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

func init() {
	CommandMap["likeme"] = likeme
	CommandMap["likethat"] = likethat
	CommandMap["likepost"] = likepost
}


func likeme(extraArgs string, splitArgs []string, c *CommandContext) {
	c.Bot.LikePost(c.Post.Id)
	log.Info("liked post", c.Post.Id, "by likeme command")
}


func likethat(extraArgs string, splitArgs []string, c *CommandContext) {
	repliedPost, err := c.Bot.GetPostByNumber(c.Post.Topic_id, c.Post.Reply_to_post_number)
	if err == nil {
		c.Bot.LikePost(repliedPost.Id)
		log.Info("liked post", repliedPost.Id, "by likethat command")
	}
}


func likepost(extraArgs string, splitArgs []string, c *CommandContext) {
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
	postToLike, err := c.Bot.GetPostByNumber(topicId, postNum)
	if err != nil {
		return
	}
	c.Bot.LikePost(postToLike.Id)
	log.Info("liked post", postToLike.Id, "by likepost command")
}
