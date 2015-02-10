package discourse

import (
	"net/url"
	"fmt"
	"strconv"

	log "github.com/riking/DisGoBot/logging"
)

func (bot *DiscourseSite) Subscribe(channel string, callback MessageBusCallback) {
	bot.messageBusCallbacks[channel] = callback
	bot.messageBusResets <- channel
}

type notificationSubscription struct {
	callback    NotificationCallback
	types       []int
}
type notifyWPostSubscription struct {
	callback    NotifyWithPostCallback
	types       []int
}

const (
	NotificationMentioned = iota
	NotificationReplied
	NotificationQuoted
	NotificationEdited
	NotificationLiked
	NotificationPrivateMessage
	NotificationPMInvite
	NotificationInviteAccepted
	NotificationPosted
	NotificationMovedPost
	NotificationLinked
	NotificationGrantedBadge
)

var NotificationTypes = map[string]int {
	"mentioned": 1,
	"replied": 2,
	"quoted": 3,
	"edited": 4,
	"liked": 5,
	"private_message": 6,
	"invited_to_private_message": 7,
	"invitee_accepted": 8,
	"posted": 9,
	"moved_post": 10,
	"linked": 11,
	"granted_badge": 12,
}

var NotificationTypesInverse map[int]string

func (bot *DiscourseSite) SubscribeNotification(callback NotificationCallback, notifyTypes []int) {
	bot.notifyCallbacks = append(bot.notifyCallbacks, notificationSubscription{callback, notifyTypes})
}

func (bot *DiscourseSite) SubscribeNotificationPost(callback NotifyWithPostCallback, notifyTypes []int) {
	bot.notifyPostCallbacks = append(bot.notifyPostCallbacks, notifyWPostSubscription{callback, notifyTypes})
}



func (bot *DiscourseSite) Login(config Config) (err error) {
	// TODO get /session/current.json
	// 404 = logged out, 200 = logged in
	err = bot.RefreshCSRF()
	if err != nil {
		return
	}

	loginData := url.Values{}
	loginData.Set("login", config.Username)
	loginData.Set("password", config.Password)
	response := ResponseUserSerializer{}

	err = bot.DPostJsonTyped("/session", loginData, &response)
	if response.User.Username == config.Username {
		log.Info("Logged in as", config.Username)
		go bot.PollNotifications(response.User.Id)

		return nil
	}
	if err != nil {
		return err
	} else {
		return ResponseGenericError{[]string{"Login failed"}, "login_failed"}
	}
}

func (d *DiscourseSite) LikePost(postId int) (err error) {
	//	d.likeRateLimit <- true
	likeData := url.Values{}
	likeData.Set("id", strconv.Itoa(postId))
	likeData.Set("post_action_type_id", "2")
	likeData.Set("flag_topic", "false")
	return d.DPost("/post_actions", likeData)
}

func (bot *DiscourseSite) ReadPosts(topicId int, posts []int) error {
	data := url.Values{}
	data.Set("topic_id", strconv.Itoa(topicId))
	data.Set("topic_time", "4242")
	for _, postId := range posts {
		data.Set(fmt.Sprintf("timings[%d]", postId), "4242")
	}
	return bot.DPost("/topics/timings", data)
}
