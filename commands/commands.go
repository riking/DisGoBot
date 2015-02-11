package commands // import "github.com/riking/DisGoBot/commands"

import (
	"strings"

	"github.com/riking/DisGoBot/discourse"
	log "github.com/riking/DisGoBot/logging"
)

type CommandRunner func(string, []string, *discourse.S_Post, *discourse.DiscourseSite)

var CommandMap = map[string]CommandRunner {}

func HasCommand(commandName string) bool {
	commandName = strings.ToLower(commandName)
	_, found := CommandMap[commandName]
	return found
}

func RunCommand(commandName string, extraArgs string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Info("Processing command", commandName, "with args", extraArgs)
	commandName = strings.ToLower(commandName)
	splitArgs := strings.Split(extraArgs, " ")

	CommandMap[commandName](extraArgs, splitArgs, post, bot)
}

func help(extraArgs string, splitArgs []string, post *discourse.S_Post, bot *discourse.DiscourseSite) {
	log.Warn("Help command not implemented")
}
