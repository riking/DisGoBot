package discourse

import (
	"encoding/gob"
	"fmt"
	//	"net/http"
	"net/http/cookiejar"
	"os"
)

type Config struct {
	Url       string
	BotName   string
}

func init() {
	var dummyJar cookiejar.Jar
	gob.Register(dummyJar)
}

// DiscourseSite

type DiscourseSite struct {
	baseUrl       string
	name          string
	cookieJar     cookiejar.Jar
}

func NewDiscourseSite(config Config) (ret *DiscourseSite, err error) {
	ret = new(DiscourseSite)
	ret.baseUrl = config.Url
	ret.name = config.BotName
	err = ret.loadCookies()

	return
}

func (d *DiscourseSite) cookieFile() string {
	return fmt.Sprintf("%s.cookies", d.name)
}

func (d *DiscourseSite) loadCookies() error {
	filename := d.cookieFile()
	file, err := os.Open(filename)
	if err != nil {
		// Make empty cookies file
		if file, err = os.Create(filename); err != nil {
			return err
		}
		return d.saveCookies()
	}
	// Load cookies
	dec := gob.NewDecoder(file)
	return dec.Decode(&d.cookieJar)
}

func (d *DiscourseSite) saveCookies() error {
	filename := d.cookieFile()
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(file)
	return enc.Encode(d.cookieJar)
}

func (d *DiscourseSite) DGet(url string) {

}

func (d *DiscourseSite) EGet(url string) {

}

