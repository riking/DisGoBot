package commands

import (
//	"strconv"
	"regexp"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
	"fmt"
)

func init() {
	CommandMap["r"] = remember
	CommandMap["rem"] = remember
	CommandMap["remember"] = remember
	CommandMap["set"] = remember

	CommandMap["recall"] = getFactoid
	CommandMap["get"] = getFactoid
	CommandMap["?"] = getFactoid
}

const rgxIdentifier = "[a-zA-z_][a-zA-Z0-9_]*"
var remember_StripName = regexp.MustCompile("\\s+" + rgxIdentifier + "\\s+(.*)")
var identifier = regexp.MustCompile(rgxIdentifier)

func remember(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	var err error
	// TODO get a more persistent store than Redis
	factoidName := splitArgs[1]

	if !identifier.MatchString(factoidName) {
		_, err = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Error: '%s' is not a valid factoid name.`, factoidName))
		log.Warn("Remember fail: Factoid name is not alphanumeric.")
		return
	}

	idxs := remember_StripName.FindStringSubmatchIndex(extraArgs)
	if idxs == nil {
		_, err = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Error: Not enough arguments.`))
		log.Warn("Remember fail: Not enough arguments.")
		return // no match
	}
	factoidBody := extraArgs[idxs[2]:]

	conn := bot.TakeUnsharedRedis()
	defer conn.Close()
	_, err = conn.Do("SET", fmt.Sprintf("disgobot:factoid:%s", factoidName), factoidBody)
	if err != nil {
		_, err = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
				`Redis error: %s`, err))
		log.Warn("Remember fail: redis error:", err)
		return
	}

	_, err = bot.Reply(post.Topic_id, post.Post_number, fmt.Sprintf(
			`Remembered '%s' as "%s".`, factoidName, factoidBody))
	log.Warn(fmt.Sprintf(`Remembered '%s' as "%s".`, factoidName, factoidBody))
}

func getFactoid(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {

}
