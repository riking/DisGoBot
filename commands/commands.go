package commands

// import "github.com/riking/DisGoBot/commands"

import (
	"github.com/garyburd/redigo/redis"
	"strings"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

type CommandRunner func(string, []string, *CommandContext)

var CommandMap = map[string]CommandRunner {}
var BannedUsers = []string{"$$nobody"}

type CommandContext struct {
	User         UserCredentials
	Post         discourse.S_Post
	Bot          *discourse.DiscourseSite
	redis        redis.Conn
	// TODO
	// Postgres  postgres.Conn
	replyBuffer  []string
}

func (cc *CommandContext) AddReply(text string) {
	cc.replyBuffer = append(cc.replyBuffer, text)
}

func (cc *CommandContext) Redis() redis.Conn {
	if cc.redis == nil {
		cc.redis = cc.Bot.TakeUnsharedRedis()
	}
	return cc.redis
}

type UserCredentials struct {
	Username           string
	UserId             int
	TrustLevel         int

	Staff              bool
	Admin              bool
	RestrictedUser     bool

	PrimaryGroup       string
}

func CredentialsFromPost(post discourse.S_Post) UserCredentials {
	return UserCredentials {
		Username: post.Username,
		UserId: post.User_id,
		TrustLevel: post.Trust_level,
		Staff: post.Staff,
		Admin: post.Admin,
		PrimaryGroup: post.Primary_group_name,
		RestrictedUser: str_contains(BannedUsers, post.Username),
	}
}

func str_contains(s []string, e string) bool {
	for _, a := range s { if a == e { return true } }
	return false
}

func HasCommand(commandName string) bool {
	commandName = strings.ToLower(commandName)
	_, found := CommandMap[commandName]
	return found
}

func RunCommand(commandName string, extraArgs string, post discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("Processing command", commandName, "with args", extraArgs)
	commandName = strings.ToLower(commandName)
	splitArgs := strings.Split(extraArgs, " ")

	var context = CommandContext{
		User: CredentialsFromPost(post),
		Post: post,
		Bot: bot,
		replyBuffer: make([]string, 2),
	}

	CommandMap[commandName](extraArgs, splitArgs, &context)

	if context.redis != nil {
		context.redis.Close()
	}
}

func help(extraArgs string, splitArgs []string, context *CommandContext) {
	log.Warn("Help command not implemented")
}

func WithRedis(bot *discourse.DiscourseSite, f func(redis.Conn)) {
	conn := bot.TakeUnsharedRedis()
	defer conn.Close()
	f(conn)
}
