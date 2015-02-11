package discourse

import (
	"fmt"
	"net/url"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"sort"
	"sync"
	"time"

	log "github.com/riking/DisGoBot/logging"
)

//import "reflect"

type SeeEveryPostCallback func(S_Post, *DiscourseSite) ()
type NotificationCallback func(S_Notification, *DiscourseSite) ()
type NotifyWithPostCallback func(S_Notification, S_Post, *DiscourseSite) ()
type MessageBusCallback func(S_MessageBus, *DiscourseSite) ()

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)

const (
	keyMessageBus  = "disgobot:MessageBusStatus"
	keyHighestPost = "disgobot:HighestPostId"
)

func (bot *DiscourseSite) pollMessageBus() {
	var postData url.Values = url.Values{}
	var pollUrl string = fmt.Sprintf("/message-bus/%s/poll", bot.clientId)
	var messageChan chan S_MessageBus = make(chan S_MessageBus)
	// messageBusPosition and positionDirty are protected by positionLock
	var messageBusPosition map[string]int = make(map[string]int)
	var positionDirty bool = false
	var positionLock sync.Mutex

	//	bot.messageBusCallbacks["/__status"] = updateChannels

	// Dispatcher thread
	go _dispatchMessageBus(messageChan, messageBusPosition, &positionLock, &positionDirty, bot)
	go _processResets(bot.messageBusResets, messageBusPosition, &positionLock)

	restoreState := func(conn redis.Conn) {
		reply, err := conn.Do("HGETALL", keyMessageBus)
		if err != nil {
			log.Error("restoring message bus state", err)
			return
		}
		if rErr, ok := reply.(redis.Error); ok {
			log.Warn("No message bus state in Redis:", rErr)
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
		log.Info("Message bus after restoring state", messageBusPosition)
		positionLock.Unlock()
	}

	// Wait for registrations
	for len(bot.messageBusCallbacks) == 0 {
		log.Debug("waiting for message bus subscribers")
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
		positionDirty = false
		positionLock.Unlock()

		if len(dataCopy) == 0 {
			return
		}

		for k, v := range dataCopy {
			if err := conn.Send("HSET", keyMessageBus, k, v); err != nil {
				log.Error(err)
			}
		}
		if err := conn.Flush(); err != nil {
			log.Error(err)
		}
		for _, _ = range dataCopy {
			if _, err := conn.Receive(); err != nil {
				log.Error(err)
			}
		}
		err := conn.Close()
		if err != nil {
			log.Error("Persisting message bus state to Redis:", err)
		} else {
			log.Info("Persisted message bus state to Redis")
		}
	}

	for {
		var response ResponseMessageBus
		// Set up form data
		postData = url.Values{}
		positionLock.Lock()
		for channel, pos := range messageBusPosition {
			// read and copy
			postData.Set(channel, strconv.Itoa(pos))
		}
		positionLock.Unlock()

		// Send request
		err := bot.DPostJsonTyped(pollUrl, postData, &response)
		if err != nil {
			log.Error("Polling message bus", err)
			time.Sleep(60 * time.Second)
			continue
		}

		if len(response) > 0 {
			log.Debug("Message bus response", response)
		}

		// Dump into channel
		for _, msg := range response {
			messageChan <- msg
		}

		time.Sleep(3 * time.Second)

		if func() bool {
			positionLock.Lock()
			defer positionLock.Unlock()
			return positionDirty
		}() {
			saveState()
		}
	}
}

func _dispatchMessageBus(messageChan chan S_MessageBus,
	messageBusPosition map[string]int,
	positionLock *sync.Mutex,
	positionDirty *bool,
	bot *DiscourseSite) {
	for msg := range messageChan {
		positionLock.Lock()
		if msg.Channel != "/__status" {
			messageBusPosition[msg.Channel] = msg.Message_Id
			*positionDirty = true
		} else {
			for channel, pos := range msg.Data {
				messageBusPosition[channel] = int(pos.(float64))
				*positionDirty = true
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

func (bot *DiscourseSite) PollNotifications() {
	var response ResponseNotifications
	var post S_Post
	var lastSeen time.Time = time.Unix(0, 0)
	var newLastSeen time.Time = time.Unix(0, 0)

	for {
		<-bot.onNotification
		log.Info("Fetching notifications")
		err := bot.DGetJsonTyped("/notifications.json?silent=true", &response)
		if err != nil {
			log.Error("Notifications error!", err)
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

		log.Info("Got", toProcessCount, "notifications to process")
		// Mark all as read and ignore the reflection updates
		if toProcessCount > 0 {
			err = bot.DPut("/notifications/reset-new", "")
			if err != nil {
				log.Error("Notifications error!", "reset-new", err)
			} else {
				<-bot.onNotification
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
			log.Info("Processing notification at", notification.Created_at_ts)
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
					log.Info("Fetching post ", notification.Topic_id, notification.Post_number, "from notification")
					err = bot.DGetJsonTyped(fmt.Sprintf("/posts/by_number/%d/%d.json", notification.Topic_id, notification.Post_number), &post)
					if err != nil {
						log.Error("Notifications error!", err)
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

		log.Info("Finished processing", processedNum, "notifications")

		time.Sleep(2 * time.Second)
	}
}

func notificationsChannel(msg S_MessageBus, bot *DiscourseSite) {
	if msg.Data["total_unread_notifications"].(float64) > 0 {
		bot.onNotification <- true
	} else {
		log.Debug("Ignoring superflous /notifications message id", msg.Message_Id)
	}
}

func (bot *DiscourseSite) PollLatestPosts() {
	var highestSeen int
	var postChannel chan S_Post = make(chan S_Post)

	restoreState := func() {
		conn := bot.GetSharedRedis()
		reply, err := conn.Do("GET", keyHighestPost)
		bot.ReleaseSharedRedis(conn)
		if err != nil {
			log.Error("Restoring post polling state:", err)
			return
		}
		if reply == nil {
			log.Warn("No post polling state to restore")
			return
		}
		if numStr, ok := reply.([]uint8); ok {
			num, err := strconv.Atoi(string(numStr))
			if err != nil {
				log.Error("post polling restore error: not an integer", err)
			} else {
				highestSeen = num
				log.Info("Restored post polling at post ID", highestSeen)
			}
		} else {
			log.Error("Bad type in redis reply? (post polling)", reply)
		}
	}

	saveState := func(highestSeen int) {
		conn := bot.GetSharedRedis()
		reply, err := conn.Do("SET", keyHighestPost, strconv.Itoa(highestSeen))
		bot.ReleaseSharedRedis(conn)

		if err != nil {
			log.Error("Saving post polling state:", err)
		}
		if okStr, ok := reply.(string); ok && okStr == "OK" {
			log.Info("Persisted post polling state in Redis")
		} else {
			log.Error("Bad type in redis reply? (post polling)", reply)
		}
	}
	_ = saveState


	restoreState()
	go _dispatchLatestPosts(postChannel, bot)
	if highestSeen == 0 {
		_seen, err := _doFirstBatch(postChannel, bot)
		if err != nil {
			log.Error("!!!! Could not load first batch of posts - cancelling post polling")
			return
		}
		highestSeen = _seen
	}

	for {
		var response ResponseLatestPosts
		var dirty = false

		select {
		case <-bot.PostHappened:
		case <-time.After(1 * time.Minute):
		}

		log.Debug("Polling for latest posts")
		err := bot.DGetJsonTyped(fmt.Sprintf("/posts.json?before=%d", highestSeen + 50), &response)
		if err != nil {
			log.Error("Error polling for latest posts:", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// reverse iterate
		for i := len(response.Latest_posts) - 1; i >= 0; i-- {
			post := response.Latest_posts[i]
			if post.Id > highestSeen {
				highestSeen = post.Id
				dirty = true
			}

			postChannel <- post
		}

		if (dirty) {
			saveState(highestSeen)
		}
	}
}

func _doFirstBatch(postChan chan<- S_Post, bot *DiscourseSite) (highestPost int, err error) {
	var response ResponseLatestPosts

	err = bot.DGetJsonTyped("/posts.json", &response)
	if err != nil {
		return -1, err
	}

	highestPost = 0
	// reverse iterate
	for i := len(response.Latest_posts) - 1; i >= 0; i-- {
		post := response.Latest_posts[i]
		if post.Id > highestPost {
			highestPost = post.Id
		}
		postChan <- post
	}
	log.Debug("Highest post ID on first check was", highestPost)
	return highestPost, nil
}

func _dispatchLatestPosts(postChan <-chan S_Post,
	bot *DiscourseSite) {
	for post := range postChan {
		log.Debug(fmt.Sprintf("Dispatching post {id %d topic %d num %d}", post.Id, post.Topic_id, post.Post_number))
		for _, handler := range bot.everyPostCallbacks {
			// TODO filters, extra context (topic? category?)
			handler.channel <- post
		}
	}
}

/*
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
			log.Error("failed to load /posts.json", err)
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
*/
