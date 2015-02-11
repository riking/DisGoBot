package discourse // import "github.com/riking/DisGoBot/discourse"

import (
	"encoding/gob"
	"fmt"
	"encoding/hex"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"crypto/rand"
	"github.com/garyburd/redigo/redis"
	"regexp"
	"strconv"
	"sync"
	"time"

	log "github.com/riking/DisGoBot/logging"
)

const VERSION = "0.2"

type Config struct {
	Url       string
	BotName   string

	Username  string
	Password  string

	RedisURL         string
	RedisDB          int
	RedisTimeoutSecs float64
}

type ConfigError string

func (e ConfigError) Error() string { return string(e); }

func init() {
	var dummyJar cookiejar.Jar
	gob.Register(dummyJar)

	NotificationTypesInverse = make(map[int]string)
	for k, v := range NotificationTypes {
		NotificationTypesInverse[v] = k
	}
}

// DiscourseSite

type DiscourseSite struct {
	// Config strings
	baseUrl       string
	name          string
	Username      string

	// Generated strings
	clientId      string
	csrfToken     string
	userId		  int

	// Client objects
	cookieJar            *cookiejar.Jar
	httpClient           http.Client
	redisPool            redis.Pool
	_sharedRedisConn     redis.Conn
	_sharedRedisRefcount int
	_sharedRedisLock     sync.Mutex

	// Channels
	rateLimit        chan *http.Request
	likeRateLimit    chan bool
	onNotification   chan bool
	messageBusResets chan string
	PostHappened     chan bool

	// Callback holders
	messageBusCallbacks   map[string]MessageBusCallback
	notifyCallbacks       []notificationSubscription
	notifyPostCallbacks   []notifyWPostSubscription
	everyPostCallbacks    []notifyEveryPostSubscription
}

// TODO this var is ugly
var onNotification chan bool

var messageBusUrlRegex = regexp.MustCompile(`/message-bus/`)

func NewDiscourseSite(config Config) (bot *DiscourseSite, err error) {
	bot = new(DiscourseSite)

	bot.baseUrl = config.Url
	bot.name = config.BotName
	bot.Username = config.Username

	bot.cookieJar, _ = cookiejar.New(nil) // never errors
	bot.httpClient.Jar = bot.cookieJar
	var redisDB = strconv.Itoa(config.RedisDB)
	bot.redisPool = redis.Pool {
		MaxIdle: 2,
		MaxActive: 5,
		Dial: func() (redis.Conn, error) {
			client, e := redis.Dial("tcp", config.RedisURL)
			if e != nil {
				return nil, e
			}
			r, e := client.Do("SELECT", redisDB)
			if e != nil {
				client.Close()
				return nil, e
			}
			if selectErr, typeCheck := r.(redis.Error); typeCheck {
				client.Close()
				return nil, selectErr
			}
			return client, nil
		},
		Wait: true,
		// casting to assure that I do want float multiply casted to int
		IdleTimeout: time.Duration(int64(config.RedisTimeoutSecs * float64(time.Second))),
	}

	bot.rateLimit = make(chan *http.Request)
	bot.likeRateLimit = make(chan bool)
	bot.onNotification = make(chan bool)
	onNotification = bot.onNotification
	bot.messageBusResets = make(chan string, 10) // TODO HACK HACK HACK
	bot.PostHappened = make(chan bool)

	bot.messageBusCallbacks = make(map[string]MessageBusCallback)
	bot.clientId = uuid()

	err = bot.loadCookies()
	if err != nil {
		return nil, err
	}

	// Feed ratelimit
	go func() {
		var req *http.Request
		for {
			time.Sleep(3 * time.Second)
			req = <-bot.rateLimit
			if !messageBusUrlRegex.MatchString(req.URL.String()) {
				log.Info("Made request to", req.URL)
			}
			req = <-bot.rateLimit
			if !messageBusUrlRegex.MatchString(req.URL.String()) {
				log.Info("Made request to", req.URL)
			}
			req = <-bot.rateLimit
			if !messageBusUrlRegex.MatchString(req.URL.String()) {
				log.Info("Made request to", req.URL)
			}
		}
	}()
	go func() {
		for {
			for i := 0; i < (450/24); i++ {
				<-bot.likeRateLimit
			}
			log.Warn("Exhausted hourly like limit")
			time.Sleep(1 * time.Hour)
		}
	}()


	return bot, nil
}

func (bot *DiscourseSite) Start() {
	// all subscriptions must be before Start()... so insert one here
	bot.Subscribe(fmt.Sprintf("/notification/%d", bot.userId), notificationsChannel)

	go bot.pollMessageBus()
	go bot.PollNotifications()
	go bot.PollLatestPosts()
	onNotification <- true
}

// A DiscourseSite instance is not safe to use after being destroyed.
func (bot *DiscourseSite) Destroy() (err error) {
	err2 := bot.redisPool.Close()

	if err2 != nil {
		return err2
	}
	return nil
}

func uuid() string {
	u := make([]byte, 16)
	_, err := rand.Read(u)
	if err != nil {
		return "123456789abcdef"
	}

	u[8] = (u[8]|0x80)&0xBF
	u[6] = (u[6]|0x40)&0x4F
	return hex.EncodeToString(u)
}

func (bot *DiscourseSite) GetSharedRedis() redis.Conn {
	bot._sharedRedisLock.Lock()
	defer bot._sharedRedisLock.Unlock()

	if bot._sharedRedisRefcount == 0 {
		bot._sharedRedisConn = bot.redisPool.Get()
		bot._sharedRedisRefcount = 1
		return bot._sharedRedisConn
	} else {
		bot._sharedRedisRefcount++
		return bot._sharedRedisConn
	}
}

func (bot *DiscourseSite) ReleaseSharedRedis(conn redis.Conn) {
	bot._sharedRedisLock.Lock()
	defer bot._sharedRedisLock.Unlock()

	if conn != bot._sharedRedisConn {
		panic("Attempt to release the wrong shared redis connection!")
	}
	if bot._sharedRedisRefcount == 1 {
		bot._sharedRedisConn.Close()
		bot._sharedRedisRefcount = 0
	} else {
		bot._sharedRedisRefcount--
	}
}

func (d *DiscourseSite) TakeUnsharedRedis() redis.Conn {
	return d.redisPool.Get()
}

func (bot *DiscourseSite) ListDomains() []string {
	return []string{bot.baseUrl}
}

func (d *DiscourseSite) cookieFile() string {
	return fmt.Sprintf("%s.cookies", d.name)
}

func (bot *DiscourseSite) loadCookies() error {
	filename := bot.cookieFile()
	file, err := os.Open(filename)
	if err != nil {
		if file != nil {
			file.Close()
		}
		// cookies are empty, first run
		return nil
	}
	defer file.Close()

	// Load cookies
	var cookies map[string][]http.Cookie
	dec := gob.NewDecoder(file)
	err = dec.Decode(&cookies)


	if err != nil {
		log.Error("Could not restore cookies:", err)
		return nil
	}
	if len(cookies) > 0 {
		for domain, cookieSlice := range cookies {
			u, urlErr := url.Parse(domain)
			if urlErr != nil {
				log.Error("Bad URL in cookie file", urlErr)
				continue
			}
			cPtrSlice := make([]*http.Cookie, len(cookieSlice))
			for idx, _ := range cookieSlice {
				cPtrSlice[idx] = &cookieSlice[idx]
			}
			bot.cookieJar.SetCookies(u, cPtrSlice)
		}
		log.Info("Restored cookies.")
	} else {
		log.Info("did not find any cookies to restore")
	}
	return nil
}

func (bot *DiscourseSite) saveCookies() error {
	filename := bot.cookieFile()
	file, err := os.Create(filename)
	if err != nil {
		log.Error("saveCookies() open error", err)
		return err
	}
	defer file.Close()

	cookies := make(map[string][]http.Cookie)
	for _, domain := range bot.ListDomains() {
		u, urlErr := url.Parse(domain)
		if urlErr != nil {
			log.Error("Bad URL in bot.ListDomains()", urlErr)
			continue
		}
		cPtrAry := bot.cookieJar.Cookies(u)
		cSlice := make([]http.Cookie, len(cPtrAry))
		for idx, val := range cPtrAry {
			cSlice[idx] = *val
		}
		cookies[domain] = cSlice
	}

	enc := gob.NewEncoder(file)

	err = enc.Encode(cookies)
	enc.Encode(1)

	if err != nil {
		log.Error("Error saving cookies:", err)
	} else {
		log.Info("Saved cookies.")
	}
	return nil
}
