package discourse // import "github.com/riking/DisGoBot/discourse"

import (
	"encoding/gob"
	"fmt"
	"encoding/hex"
	"net/http"
	"net/http/cookiejar"
	"os"
	"crypto/rand"
	"github.com/garyburd/redigo/redis"
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

	// Generated strings
	clientId      string
	csrfToken     string

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

	// Callback holders
	messageBusCallbacks   map[string]MessageBusCallback
	notifyCallbacks       []notificationSubscription
	notifyPostCallbacks   []notifyWPostSubscription
}

// TODO this var is ugly
var OnNotification chan bool

func NewDiscourseSite(config Config) (bot *DiscourseSite, err error) {
	bot = new(DiscourseSite)

	bot.baseUrl = config.Url
	bot.name = config.BotName

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

	bot.rateLimit = make(chan *http.Request, 3)
	bot.likeRateLimit = make(chan bool)
	bot.onNotification = make(chan bool)
	OnNotification = bot.onNotification
	bot.messageBusResets = make(chan string)

	bot.messageBusCallbacks = make(map[string]MessageBusCallback)
	bot.clientId = uuid()

	err = bot.loadCookies()
	if err != nil {
		return nil, err
	}

	// Feed ratelimit
	go func() {
		for {
			time.Sleep(1 * time.Second)
			req := <-bot.rateLimit
			log.Info("Made request to", req.URL)
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

	go bot.pollMessageBus()

	return bot, nil
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

func (d *DiscourseSite) cookieFile() string {
	return fmt.Sprintf("%s.cookies", d.name)
}

func (d *DiscourseSite) loadCookies() error {
	filename := d.cookieFile()
	file, err := os.Open(filename)
	if err != nil {
		file.Close()
		// cookies are empty, first run
		return nil
	}
	// Load cookies
	defer file.Close()
	dec := gob.NewDecoder(file)
	return dec.Decode(&d.cookieJar)
}

func (d *DiscourseSite) saveCookies() error {
	filename := d.cookieFile()
	file, err := os.Open(filename)
	if err != nil {
		log.Error("saveCookies() open error", err)
		return err
	}
	enc := gob.NewEncoder(file)
	err = enc.Encode(d.cookieJar)
	if err != nil {
		log.Error("encode error", err)
	}
	return nil
}
