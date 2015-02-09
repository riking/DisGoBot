package discourse

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
	"crypto/rand"
	"encoding/hex"
)

type Config struct {
	Url       string
	BotName   string

	Username  string
	Password  string
}

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
	baseUrl       string
	name          string
	cookieJar     *cookiejar.Jar
	httpClient    http.Client
	clientId      string
	csrfToken     string

	rateLimit        chan *http.Request
	likeRateLimit    chan bool
	onNotification   chan bool

	messageBus            map[string]int
	messageBusCallbacks   map[string]MessageBusCallback
	notifyCallbacks       []notificationSubscription
	notifyPostCallbacks   []notifyWPostSubscription
}

func NewDiscourseSite(config Config) (bot *DiscourseSite, err error) {
	bot = new(DiscourseSite)

	bot.baseUrl = config.Url
	bot.name = config.BotName
	bot.cookieJar, err = cookiejar.New(nil)
	bot.httpClient.Jar = bot.cookieJar

	bot.rateLimit = make(chan *http.Request)
	bot.likeRateLimit = make(chan bool)
	bot.onNotification = make(chan bool)

	bot.messageBus = make(map[string]int)
	bot.messageBusCallbacks = make(map[string]MessageBusCallback)
	bot.clientId = uuid()

	err = bot.loadCookies()

	// Feed ratelimit
	go func() {
		for {
			time.Sleep(1 * time.Second)
			req := <-bot.rateLimit
			fmt.Printf("Made request to %s\n", req.URL)
		}
	}()
	go func() {
		for {
			for i := 0; i < (500/24); i++ {
				<-bot.likeRateLimit
			}
			fmt.Println("Exhausted hourly like limit")
			time.Sleep(1 * time.Hour)
		}
	}()

	go bot.PollMessageBus()

	return
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
		fmt.Println("saveCookies() open error")
		return err
	}
	enc := gob.NewEncoder(file)
	err = enc.Encode(d.cookieJar)
	fmt.Println("encode error", err)
	return nil
}
