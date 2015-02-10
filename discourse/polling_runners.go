package discourse

//import "github.com/riking/discourse/discourse"
import (
	"fmt"
	"net/url"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"sort"
	"sync"
	"time"
)
//import "reflect"

type SeeEveryPostCallback func(S_Post, *DiscourseSite) ()
type NotificationCallback func(S_Notification, *DiscourseSite) ()
type NotifyWithPostCallback func(S_Notification, S_Post, *DiscourseSite) ()
type MessageBusCallback func(S_MessageBus, *DiscourseSite) ()

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)

const (
	keyMessageBus = "disgobot:MessageBusStatus"
)

func (bot *DiscourseSite) pollMessageBus() {
	var postData url.Values = url.Values{}
	var pollUrl string = fmt.Sprintf("/message-bus/%s/poll", bot.clientId)
	var messageChan chan S_MessageBus = make(chan S_MessageBus)
	var messageBusPosition map[string]int = make(map[string]int)
	var positionLock sync.Mutex
	var lastRedisSave time.Time = time.Now()

	//	bot.messageBusCallbacks["/__status"] = updateChannels

	// Dispatcher thread
	go _dispatchMessageBus(messageChan, messageBusPosition, &positionLock, bot)
	go _processResets(bot.messageBusResets, messageBusPosition, &positionLock)

	restoreState := func(conn redis.Conn) {
		reply, err := conn.Do("HGETALL", keyMessageBus)
		if err != nil {
			fmt.Println("[ERR]", "restoring message bus state", err)
			return
		}
		if rErr, ok := reply.(redis.Error); ok {
			fmt.Println("[WARN]", "No message bus state in Redis:", rErr)
			return
		}
		l := reply.([]interface{})
		list := make([]string, len(l))
		for i, v := range l {
			list[i] = string(v.([]uint8))
		}

		positionLock.Lock()
		for i := 0; i < len(list); i = i+2 {
			n, err := strconv.Atoi(list[i + 1])
			if n != 0 && err == nil {
				messageBusPosition[list[i]] = n
			} else {
				messageBusPosition[list[i]] = -1
			}
		}
		fmt.Println("[INFO]", "Message bus after restoring state", messageBusPosition)
		positionLock.Unlock()
	}

	// Wait for registrations
	for len(bot.messageBusCallbacks) == 0 {
		time.Sleep(1 * time.Second)
	}
	time.Sleep(1 * time.Second)

	c := bot.GetSharedRedis()
	restoreState(c)
	bot.ReleaseSharedRedis(c)

	saveState := func() {
		conn := bot.TakeUnsharedRedis()
		dataCopy := make(map[string]string)
		positionLock.Lock()
		for k, v := range messageBusPosition {
			dataCopy[k] = strconv.Itoa(v)
		}
		positionLock.Unlock()

		for k, v := range dataCopy {
			if err := conn.Send("HSET", keyMessageBus, k, v); err != nil {
				fmt.Println(err)
			}
		}
		if err := conn.Flush(); err != nil {
			fmt.Println(err)
		}
		for _, _ = range dataCopy {
			if _, err := conn.Receive(); err != nil {
				fmt.Println(err)
			}
		}
		err := conn.Close()
		if err != nil {
			fmt.Println("[ERR]", "Persisting message bus state to Redis:", err)
		}
	}

	var response ResponseMessageBus
	for {
		// Set up form data
		postData = url.Values{}
		positionLock.Lock()
		for channel, pos := range messageBusPosition {
			postData.Set(channel, strconv.Itoa(pos))
		}
		positionLock.Unlock()

		// Send request
		err := bot.DPostJsonTyped(pollUrl, postData, &response)
		if err != nil {
			fmt.Println(err)
			time.Sleep(60 * time.Second)
		}

		fmt.Println("[DBUG]", "Message bus response", response)
		// Dump into channel
		for _, msg := range response {
			messageChan <- msg
		}
		time.Sleep(3 * time.Second)

		if lastRedisSave.Add(30 * time.Second).Before(time.Now()) {
			fmt.Println("[INFO]", "Persisting message bus to Redis")
			lastRedisSave = time.Now()
			saveState()
		}
	}
}

func _dispatchMessageBus(messageChan chan S_MessageBus,
	messageBusPosition map[string]int,
	positionLock *sync.Mutex,
	bot *DiscourseSite) {
	for msg := range messageChan {
		positionLock.Lock()
		if msg.Channel != "/__status" {
			messageBusPosition[msg.Channel] = msg.Message_Id
		} else {
			for channel, pos := range msg.Data {
				messageBusPosition[channel] = int(pos.(float64))
			}
		}
		positionLock.Unlock()

		// TODO multiple callbacks on one channel
		callback := bot.messageBusCallbacks[msg.Channel]
		if callback != nil {
			callback(msg, bot)
		}
	}
}

func _processResets(toReset chan string,
	messageBusPosition map[string]int,
	positionLock *sync.Mutex) {
	for channel := range toReset {
		positionLock.Lock()
		messageBusPosition[channel] = -1
		positionLock.Unlock()
	}
}

func notificationsChannel(msg S_MessageBus, bot *DiscourseSite) {
	if msg.Data["total_unread_notifications"].(float64) > 0 {
		bot.onNotification <- true
	}
}

func contains(s []int, e int) bool {
	for _, a := range s { if a == e { return true } }
	return false
}

type ByCreatedAt ResponseNotifications

func (r ByCreatedAt) Len() int { return len(r) }
func (r ByCreatedAt) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ByCreatedAt) Less(j, i int) bool {
	return r[i].Created_at_ts.Before(r[j].Created_at_ts)
}

// note: started from Login()
func (bot *DiscourseSite) PollNotifications(userId int) {
	busChannel := fmt.Sprintf("/notification/%d", userId)
	bot.Subscribe(busChannel, notificationsChannel)

	var response ResponseNotifications
	var post S_Post
	var lastSeen time.Time = time.Unix(0, 0)
	var newLastSeen time.Time = time.Unix(0, 0)

	for {
		<-bot.onNotification
		fmt.Println("[INFO]", "Fetching notifications")
		err := bot.DGetJsonTyped("/notifications.json", &response)
		if err != nil {
			fmt.Println("[ERR]", "Notifications error!", err)
			time.Sleep(60 * time.Second)
			continue
		}

		// Sort by created_at to fix problems with bubbled notifications
		for idx, _ := range response {
			response[idx].ParseTimes()
		}
		sort.Sort(ByCreatedAt(response))

		toProcessCount := 0
		for _, n := range response {
			if !n.Read {
				toProcessCount = toProcessCount+1
			}
		}

		fmt.Println("[INFO]", "Got", toProcessCount, "notifications to process")
		// Mark all as read and ignore the reflection updates
		if toProcessCount > 0 {
			err = bot.DPut("/notifications/reset-new", "")
			if err != nil {
				fmt.Println("[ERR]", "Notifications error!", "reset-new", err)
			}
		}

		processedNum := 0
		for _, notification := range response {
			if notification.Read {
				continue
			}
			if notification.Created_at_ts.Before(lastSeen) {
				continue
			}
			if notification.Created_at_ts.After(newLastSeen) {
				newLastSeen = notification.Created_at_ts
			}
			fmt.Println("[INFO]", "Processing notification at", notification.Created_at_ts)
			processedNum++

			notifyType := notification.Notification_type

			for _, handler := range bot.notifyCallbacks {
				if contains(handler.types, notifyType) {
					handler.callback(notification, bot)
				}
			}


			// If the notification is assosciated with a post
			if notification.Topic_id > 0 {
				if len(bot.notifyPostCallbacks) > 0 {
					fmt.Println("[INFO]", "Fetching post ", notification.Topic_id, notification.Post_number, "from notification")
					err = bot.DGetJsonTyped(fmt.Sprintf("/posts/by_number/%d/%d.json", notification.Topic_id, notification.Post_number), &post)
					if err != nil {
						fmt.Println("[ERR]", "Notifications error!", err)
						time.Sleep(60 * time.Second)
					} else {
						for _, handler := range bot.notifyPostCallbacks {
							if contains(handler.types, notifyType) {
								handler.callback(notification, post, bot)
							}
						}
					}
				}
			}
		}
		lastSeen = newLastSeen

		fmt.Println("[INFO]", "Finished processing", processedNum, "notifications")

		time.Sleep(2 * time.Second)
	}
}

// TODO this is gahbage
func SeeEveryPost(bot *DiscourseSite, highestSeen *int, callback SeeEveryPostCallback, onlyBelow int) {
	var posts ResponseLatestPosts
	var request string
	var myHighest int = 0

	lowestId := MaxInt
	if onlyBelow > 0 {
		lowestId = onlyBelow
	}

	for lowestId > *highestSeen {
		if request == "" && onlyBelow <= 0 {
			request = "/posts.json" // first loop
		} else {
			request = fmt.Sprintf("/posts.json?before=%d", lowestId)
		}

		err := bot.DGetJsonTyped(request, &posts)
		if err != nil {
			fmt.Println("[ERR]", err)
			return
		}


		for _, post := range posts.Latest_posts {
			if post.Id < lowestId && post.Id > *highestSeen {
				callback(post, bot)
			}
			if post.Id > myHighest {
				myHighest = post.Id
			}
		}
		if lowestId == MaxInt {
			lowestId = posts.Latest_posts[0].Id // not optimal
		} else {
			lowestId = lowestId-50
		}
	}
	*highestSeen = myHighest
}
