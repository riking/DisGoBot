package discourse

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
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
}

// DiscourseSite

type DiscourseSite struct {
	baseUrl       string
	name          string
	cookieJar     *cookiejar.Jar
	rateLimit     chan *http.Request
	likeRateLimit chan bool
	httpClient    http.Client

	csrfToken string
}

func NewDiscourseSite(config Config) (bot *DiscourseSite, err error) {
	bot = new(DiscourseSite)

	bot.baseUrl = config.Url
	bot.name = config.BotName
	bot.cookieJar, err = cookiejar.New(nil)
	bot.rateLimit = make(chan *http.Request)
	bot.likeRateLimit = make(chan bool)

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
			for i := 0; i < (500 / 24); i++ {
				<-bot.likeRateLimit
			}
			fmt.Println("Exhausted hourly like limit")
			time.Sleep(1 * time.Hour)
		}
	}()
	bot.httpClient.Jar = bot.cookieJar

	return
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
