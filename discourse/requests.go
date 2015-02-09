package discourse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/riking/discourse/meta"
)

// Repsonse Types
type ResponseCSRF struct {
	Csrf string
}
type ResponseGenericError struct {
	errors     []string
	error_type string
}

// Error types
type ErrorWithJSON struct {
	Status int
	Json map[string]interface{}
}
type ErrorRateLimit struct {
	string   string
}
type ErrorNotFound bool
type ErrorPermissions bool
type ErrorReadOnly bool
type ErrorBadCsrf bool
type ErrorStatusCode int
type ErrorEmptyResponse bool
type ErrorBadJsonType struct {
	Child error
	Json string
}

func (e ResponseGenericError) Error() string {
	return strings.Join(e.errors, "; ")
}
func (e ErrorWithJSON) Error() string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.Encode(e.Json)
	jsonString := b.String()
	// Truncate the string
	if len(jsonString) > 200 {
		jsonString = jsonString[:200]
	}
	return fmt.Sprintf("Server returned status code %d: %s", e.Status, jsonString)
}
func (e ErrorRateLimit) Error() string {
	return "Rate limit exceeded"
}
func (e ErrorPermissions) Error() string {
	return "Invalid access"
}
func (e ErrorReadOnly) Error() string {
	return "Site is in read-only mode"
}
func (e ErrorNotFound) Error() string {
	return "Server responsded with 404 Not Found"
}
func (e ErrorBadCsrf) Error() string {
	return "Server responded with Bad CSRF error"
}
func (e ErrorBadJsonType) Error() string {
	return fmt.Sprintf("Bad json type: %s\nJson: %s", e.Child.Error(), e.Json)
}
func (e ErrorStatusCode) Error() string {
	return fmt.Sprintf("Server returned status code %d", e)
}
func (e ErrorEmptyResponse) Error() string {
	return "Server produced an empty response"
}

// Methods

func (d *DiscourseSite) addBase(url string) string {
	return fmt.Sprintf("%s%s", d.baseUrl, url)
}

func addHeaders(d *DiscourseSite, req *http.Request) {
	req.Header = map[string][]string {
		"Accept-Language": {"en-us"},
		"Connection": {"keep-alive"},
		"User-Agent": {fmt.Sprintf("DisGoBot %s @%s", meta.VERSION, d.name)},
		"X-Requested-With": {"XMLHttpRequest"},
		"X-CSRF-Token": {d.csrfToken},
	}
}

// Execute a request, obeying the ratelimit
func (d *DiscourseSite) do(req *http.Request) (resp *http.Response, err error) {
	d.rateLimit <- req
	return d.httpClient.Do(req)
}

// Public alias for do() method
func (d *DiscourseSite) PerformRequest(req *http.Request) (resp *http.Response, err error) {
	return d.do(req)
}

func (d *DiscourseSite) RefreshCSRF() (err error) {
	if d.csrfToken != "" {
		panic("unneeded refreshing of csrf")
	}
	req, err := d.makeRequest("/session/csrf.json")
	if err != nil {
		return
	}
	resp, err := d.do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode > 200 {
		return ErrorStatusCode(resp.StatusCode)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var csrf ResponseCSRF
	err = json.Unmarshal(buf, &csrf)
	if err != nil {
		fmt.Println("error getting csrf", err, buf)
		return err
	}

	d.csrfToken = csrf.Csrf
	return nil
}

// Create a http.Request for GET url
func (d *DiscourseSite) makeRequest(url string) (req *http.Request, err error) {
	req, err = http.NewRequest("", d.addBase(url), nil)
	addHeaders(d, req)
	return req, err
}
func (d *DiscourseSite) makeRequestPost(url string, data url.Values) (req *http.Request, err error) {
	req, err = http.NewRequest("POST", d.addBase(url), bytes.NewBufferString(data.Encode()))
	addHeaders(d, req)
	return req, err
}

var haltRecursion = false

// Read JSON from a Discourse endpoint into a typed variable.s
func (d *DiscourseSite) DGetJsonTyped(url string, target interface{}) (err error) {
	req, err := d.makeRequest(url)
	if err != nil {
		return err
	}
	req.Header["Accept"] = []string{"application/json, text/javascript"}

	err = d.decodeJsonTyped(req, target)
	return
}

func (d *DiscourseSite) decodeJsonTyped(request *http.Request, target interface{}) (err error) {
	if d.csrfToken == "" {
		err = d.RefreshCSRF()
		if err != nil {
			return err
		}
	}

	resp, err := d.do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	asString := string(buf)
	if asString == "['BAD CSRF']" {
		d.csrfToken = ""
		d.RefreshCSRF()
		return ErrorBadCsrf(false)
	}

	marshErr := json.Unmarshal(buf, target)
	if marshErr != nil {
		if resp.StatusCode >= 400 {
			// TODO - special behavior for some of these?
			switch(resp.StatusCode) {
			case 403:
				return ErrorPermissions(false)
			case 404:
				return ErrorNotFound(false)
			case 405:
				return ErrorReadOnly(false)
			case 429:
				return ErrorRateLimit{asString}
			}
			var dError ErrorWithJSON
			marshErr2 := json.Unmarshal(buf, dError)
			if marshErr2 == nil {
				return dError
			}
			return ErrorStatusCode(resp.StatusCode)
		} else {
			return ErrorBadJsonType{marshErr, asString}
		}
	}
	return nil
}

// Read JSON from a Discourse endpoint into a generic map.
func (d *DiscourseSite) DGetJson(url string) (response map[string]interface{}, err error) {
	response = make(map[string]interface{})
	err = d.DGetJsonTyped(url, response)
	return
}

func (d *DiscourseSite) DPost(url string, data url.Values) (err error) {
	req, err := d.makeRequestPost(url, data)
	if err != nil {
		return err
	}
	req.Header["Accept"] = []string{"application/json, text/javascript"}

	_, err = d.do(req)
	return
}

func (d *DiscourseSite) DPostJsonTyped(url string, data url.Values, target interface{}) (err error) {
	req, err := d.makeRequestPost(url, data)
	if err != nil {
		return err
	}
	req.Header["Accept"] = []string{"application/json, text/javascript"}

	err = d.decodeJsonTyped(req, target)
	return
}
