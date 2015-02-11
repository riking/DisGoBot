package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

func init() {
	CommandMap["likeme"] = likeme
	CommandMap["likethat"] = likethat
	CommandMap["likepost"] = likepost

	CommandMap["seen"] = seen
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
	if len(splitArgs) < 2+1 {
		c.AddReply("Not enough arguments.")
		return
	}
	topicId, err := strconv.Atoi(splitArgs[1])
	if err != nil {
		c.AddReply(fmt.Sprintf(
			"Cannot parse %s as number: %s", splitArgs[1], err))
		return
	}
	postNum, err := strconv.Atoi(splitArgs[2])
	if err != nil {
		c.AddReply(fmt.Sprintf(
			"Cannot parse %s as number: %s", splitArgs[2], err))
		return
	}
	postToLike, err := c.Bot.GetPostByNumber(topicId, postNum)
	if err != nil {
		return
	}
	c.Bot.LikePost(postToLike.Id)
	log.Info("liked post", postToLike.Id, "by likepost command")
}

var usernameRegex = regexp.MustCompile("^[a-zA-Z][a-zA-z0-9_]+$")
func seen(extraArgs string, splitArgs []string, c *CommandContext) {
	if len(splitArgs) < 1+1 {
		c.AddReply("Not enough arguments.")
		return
	}
	log.Debug(strings.Join(splitArgs,","))
	username := splitArgs[1]
	if !usernameRegex.MatchString(username) {
		c.AddReply(fmt.Sprintf(
			"'%s' is not a valid username.", username))
		return
	}

	var response discourse.ResponseUserSerializer

	err := c.Bot.DGetJsonTyped(fmt.Sprintf("/users/%s.json", username), &response)
	if err != nil {
		if _, ok := err.(discourse.ErrorNotFound); ok {
			c.AddReply(fmt.Sprintf(
				"The user '%s' does not exist.", username))
		} else {
			c.AddReply(fmt.Sprintf(
				"Error fetching data: " + err.Error()))
		}
		return
	}

	response.User.ParseTimes()

	duration := time.Since(response.User.LastSeenAtTime)
	duration = duration % time.Second
	c.AddReply(fmt.Sprintf(
		"@%s was last seen on the date %s, which was %s ago.",
		response.User.Username, response.User.LastSeenAtTime.Format(time.Stamp), duration))
}
