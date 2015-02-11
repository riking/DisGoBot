package commands

import (
	//	"strconv"
	"github.com/garyburd/redigo/redis"
	"regexp"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
	"fmt"
)

var FactoidHandlers = map[string]FactoidHandlerFunc{}

func init() {
	CommandMap["r"] = remember
	CommandMap["rem"] = remember
	CommandMap["remember"] = remember
	CommandMap["set"] = remember

	CommandMap["recall"] = cmdGetFactoid
	CommandMap["get"] = cmdGetFactoid
	CommandMap["?"] = cmdGetFactoid

	FactoidHandlers["alias"] = factoidHandlerAlias
	FactoidHandlers["reply"] = factoidHandlerReply
}

// First string is factoid raw, second string is arguments
type FactoidHandlerFunc func(redis.Conn, string, string, *discourse.S_Post, *discourse.DiscourseSite) (string, error)
type FactoidError string
func (e FactoidError) Error() string { return string(e) }

const rgxFactoidName = "[a-zA-Z0-9?!_-]+"
const rgxHandlerName = "[a-z]+"

var remember_StripName = regexp.MustCompile("\\s+" + rgxFactoidName + "\\s+(.*)")
var factoidPattern = regexp.MustCompile(rgxFactoidName)
var handlerPattern = regexp.MustCompile("<(" + rgxHandlerName + ")>")


func remember(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	var err error
	// TODO get a more persistent store than Redis
	factoidName := splitArgs[1]

	if !factoidPattern.MatchString(factoidName) {
		_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Error: '%s' is not a valid factoid name.`, factoidName))
		log.Warn("Remember fail: Factoid name is not alphanumeric.")
		return
	}

	idxs := remember_StripName.FindStringSubmatchIndex(extraArgs)
	if idxs == nil {
		_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
			`Error: Not enough arguments.`))
		log.Warn("Remember fail: Not enough arguments.")
		return // no match
	}
	factoidBody := extraArgs[idxs[2]:]

	conn := bot.TakeUnsharedRedis()
	defer conn.Close()
	_, err = conn.Do("SET", fmt.Sprintf("disgobot:factoid:%s", factoidName), factoidBody)
	if err != nil {
		_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Redis error: %s`, err))
		log.Warn("Remember fail: redis error:", err)
		return
	}

	_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
			`Remembered '%s' as "%s".`, factoidName, factoidBody))
	log.Warn(fmt.Sprintf(`Remembered '%s' as "%s".`, factoidName, factoidBody))
}

func cmdGetFactoid(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	var err error
	// TODO get a more persistent store than Redis
	factoidName := splitArgs[1]

	if !factoidPattern.MatchString(factoidName) {
		_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Error: '%s' is not a valid factoid name.`, factoidName))
		log.Warn("Get fail: Factoid name is not alphanumeric.")
		return
	}

	var factoidArgs string
	idxs := remember_StripName.FindStringSubmatchIndex(extraArgs)
	if idxs == nil {
		factoidArgs = ""
	} else {
		factoidArgs = extraArgs[idxs[2]:]
	}

	var response string
	WithRedis(bot, func(conn redis.Conn) {
			response, err = doFactoid(conn, factoidName, factoidArgs, post, bot)
		})

	if err != nil {
		_, _ = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
			`Factoid error: %s`, err))
		log.Warn("Factoid error:", err)
		return
	}

	_, err = bot.Reply(post.Topic_id, post.Post_number, response)
}

func doFactoid(conn redis.Conn,
	factoidName string,
	factoidArgs string,
	post *discourse.S_Post,
	bot *discourse.DiscourseSite) (result string, err error) {

	var raw string

	rawBytes, err := conn.Do("GET", fmt.Sprintf("disgobot:factoid:%s", factoidName))
	if err != nil {
		return "", err
	}
	raw = string(rawBytes.([]uint8))

	// TODO processing goes here

	if handlerPattern.MatchString(raw) {
		idxs := handlerPattern.FindStringSubmatchIndex(raw)
		if idxs == nil {
			panic("inconsistency with MatchString vs FindStringSubmatchIndex?")
		}
		handlerName := raw[idxs[2]:idxs[3]]
		handler, ok := FactoidHandlers[handlerName]
		if !ok {
			return "", FactoidError("Could not find handler called " + handlerName)
		}
		raw, err = handler(conn, raw[idxs[1]:], factoidArgs, post, bot)
	}

	result = raw
	return
}

/*
func factoidHandlerReply(conn redis.Conn,
	factoidRaw string,
	factoidArgs string,
	post *discourse.S_Post,
	bot *discourse.DiscourseSite)
*/

func factoidHandlerReply(_ redis.Conn,
	factoidRaw string,
	_ string,
	_ *discourse.S_Post,
	_ *discourse.DiscourseSite) (string, error) {
	return factoidRaw, nil
}

// any number of spaces, then non-spaces, then spaces again
var patternFirstWord = regexp.MustCompile("\\s*(" + rgxFactoidName + ")\\s*")
func factoidHandlerAlias(conn redis.Conn,
	factoidRaw string,
	_ string,
	post *discourse.S_Post,
	bot *discourse.DiscourseSite) (string, error) {

	idxs := patternFirstWord.FindStringSubmatchIndex(factoidRaw)
	if idxs == nil {
		return "", FactoidError("Alias: Nothing specified to alias to, or not a valid factoid name")
	}

	aliasedFactoidName := factoidRaw[idxs[2]:idxs[3]]
	aliasedFactoidArgs := factoidRaw[idxs[1]:]
	// TODO catch infinite recursion
	return doFactoid(conn, aliasedFactoidName, aliasedFactoidArgs, post, bot)
}
