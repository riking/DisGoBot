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
	httpClient    http.Client

	csrfToken string
}

func NewDiscourseSite(config Config) (ret *DiscourseSite, err error) {
	ret = new(DiscourseSite)

	ret.baseUrl = config.Url
	ret.name = config.BotName
	ret.cookieJar, err = cookiejar.New(nil)
	ret.rateLimit = make(chan *http.Request)

	err = ret.loadCookies()
	// Feed ratelimit
	go func() {
		for {
			time.Sleep(1 * time.Second)
			req := <-ret.rateLimit
			fmt.Printf("Made request to %s\n", req.URL)
		}
	}()
	ret.httpClient.Jar = ret.cookieJar

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
