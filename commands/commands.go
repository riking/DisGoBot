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
var IgnoredGroups = []string{"bots"}
var BotControllers = []string{"riking"}

type CommandContext struct {
	User         UserCredentials
	Post         discourse.S_Post
	Bot          *discourse.DiscourseSite
	redis        redis.Conn
	// TODO
	// Postgres  postgres.Conn

	replyBuffer  []string

	// A string buffer, to be used for recursion detection.
	RecursionChain []string
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
	Controller         bool

	PrimaryGroup       string

	EffectiveAccessLevel int
}

func CredentialsFromPost(post discourse.S_Post) UserCredentials {
	uc := UserCredentials {
		Username: post.Username,
		UserId: post.User_id,
		TrustLevel: post.Trust_level,
		Staff: post.Staff,
		Admin: post.Admin,
		Controller: str_contains(BotControllers, post.Username),
		PrimaryGroup: post.Primary_group_name,
	}
	uc.EffectiveAccessLevel = uc._calcAccessLevel()
	return uc
}

func (uc UserCredentials) _calcAccessLevel() int {
	if str_contains(BannedUsers, uc.Username) {
		return -2
	} else if str_contains(IgnoredGroups, uc.PrimaryGroup) {
		return -1
	} else if uc.Controller {
		return 10
	} else if uc.Admin {
		return 9
	} else if uc.Staff {
		return 8
	} else {
		return uc.TrustLevel + 1
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

func RunCommandBatch(commandLines [][]string, post discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("Processing commands in post", post.Topic_id, post.Post_number, commandLines)

	var context = CommandContext{
		User: CredentialsFromPost(post),
		Post: post,
		Bot: bot,
		replyBuffer: make([]string, 0),
	}

	for _, command := range commandLines {
		log.Debug(command[1], "X", command[2])
		if HasCommand(command[1]) {
			RunCommand(command[1], command[2], &context)
		} else {
			log.Warn("No such command", command[1])
		}
	}

	if context.redis != nil {
		context.redis.Close()
	}
	if len(context.replyBuffer) > 0 {
		bot.Reply(post.Topic_id, post.Post_number, strings.Join(context.replyBuffer, "\n\n"))
	}
}

func RunCommand(commandName string, extraArgs string, context *CommandContext) {
	log.Info("Processing command", commandName, "with args", extraArgs)
	commandName = strings.ToLower(commandName)
	splitArgs := strings.Fields(extraArgs)

	CommandMap[commandName](extraArgs, splitArgs, context)
}

func help(extraArgs string, splitArgs []string, context *CommandContext) {
	log.Warn("Help command not implemented")
}

func WithRedis(bot *discourse.DiscourseSite, f func(redis.Conn)) {
	conn := bot.TakeUnsharedRedis()
	defer conn.Close()
	f(conn)
}
