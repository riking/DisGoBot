package discourse

import (
	"net/url"
	"fmt"
)



func (d *DiscourseSite) Login(config Config) (err error) {
	err = d.RefreshCSRF()
	if err != nil {
		return
	}

	loginData := url.Values{}
	loginData.Set("login", config.Username)
	loginData.Set("password", config.Password)
	response := ResponseUserSerializer{}

	err = d.DPostJsonTyped("/session", loginData, &response)
	if response.User.Username == config.Username {
		fmt.Printf("Logged in as %s\n", config.Username)
		return nil
	}
	if err != nil {
		return err
	} else {
		return ResponseGenericError{[]string{"Login failed"}, "login_failed"}
	}
}

func (d *DiscourseSite) LikePost(postId int) (err error) {
	d.likeRateLimit <- true
	likeData := url.Values{}
	likeData.Set("id", fmt.Sprintf("%d", postId))
	likeData.Set("post_action_type_id", "2")
	likeData.Set("flag_topic", "false")
	return d.DPost("/post_actions", likeData)
}
