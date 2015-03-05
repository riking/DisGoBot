package discourse

import "time"

type S_BasicUser struct {
	Id                 int
	Username           string
	Uploaded_avatar_id int
	Avatar_template    string
}
type S_UserBadge struct {
	Id            int
	Granted_at    string
	Badge_id      int
	User_id       int
	Granted_by_id int
}
type S_Badge struct {
	Id                int
	Name              string
	Description       string
	Grant_count       int
	Allow_title       bool
	Multiple_grant    bool
	Icon              string
	Image             string
	Listable          bool
	Enabled           bool
	Badge_grouping_id int
	System            bool
	Badge_type_id     int
}
type S_BadgeType struct {
	Id         int
	Name       string
	Sort_order int
}
type S_UserActionStat struct {
	Action_type int
	Count       int
}
type S_UserCustomGroup struct {
	Id                                 int
	Automatic                          bool
	Name                               string
	User_count                         int
	Alias_level                        interface{}
	Visible                            bool
	Automatic_membership_email_domains []string
	Automatic_membership_retroactive   bool
}
type S_UserProfilePrivateData struct {
	Locale                            string
	Email_digests                     bool
	Email_private_messages            bool
	Email_direct                      bool
	Email_always                      bool
	Digest_after_days                 int
	Mailing_list_mode                 bool
	Auto_track_topics_after_msecs     int
	New_topic_duration_minutes        int
	External_links_in_new_tab         bool
	Dynamic_favicon                   bool
	Enable_quoting                    bool
	Muted_category_ids                []int
	Tracked_category_ids              []int
	Watched_category_ids              []int
	Private_messages_stats            struct { all int; mine int; unread int; }
	Disable_jump_reply                bool
	Gravatar_avatar_upload_id         int
	Custom_avatar_upload_id           int
}
type S_UserProfileStaffData struct {
	Post_count                        int
	Can_be_deleted                    bool
	Can_delete_all_posts              bool
}
type S_UserProfile struct {
	S_BasicUser
	Name                              string
	Email                             string
	Last_posted_at                    string
	LastPostedAtTime                  time.Time
	Last_seen_at                      string
	LastSeenAtTime                    time.Time
	Created_at                        string
	CreatedAtTime                     time.Time
	Website                           string
	Profile_background                string
	Card_background                   string
	Location                          string
	Can_edit                          bool
	Can_edit_username                 bool
	Can_edit_email                    bool
	Can_edit_name                     bool
	Stats                             []S_UserActionStat
	Can_send_private_messages         bool
	Can_send_private_messages_to_user bool
	Bio_raw                           string
	Bio_cooked                        string
	Bio_excerpt                       string
	Trust_level                       int
	Moderator                         bool
	Admin                             bool
	Title                             *string
	Badge_count                       int
	Notification_count                int
	Has_title_badges                  bool
	Custom_fields                     map[string]interface{}
	User_fields                       map[string]string
	S_UserProfileStaffData
	S_UserProfilePrivateData
	Invited_by                        string
	Custom_groups                     []S_UserCustomGroup
	Featured_user_badge_ids           []int
	Card_badge                        interface{}
}

func (s *S_UserProfile) ParseTimes() {
	s.CreatedAtTime, _ = time.Parse(time.RFC3339, s.Created_at)
	s.LastPostedAtTime, _ = time.Parse(time.RFC3339, s.Last_posted_at)
	s.LastSeenAtTime, _ = time.Parse(time.RFC3339, s.Last_seen_at)
}

type S_Notification struct {
	Notification_type int
	Read              bool
	Created_at        string
	Created_at_ts     time.Time
	Post_number       int
	Topic_id          int
	Slug              string
	Data              map[string]interface{}
}

func (s *S_Notification) ParseTimes() {
	ts, err := time.Parse(time.RFC3339, s.Created_at)
	if err != nil {
		panic(err)
	}
	s.Created_at_ts = ts
}

type S_PostAction struct {
	Id      int
	Count   int
	Hidden  bool
	Can_act bool
}
type S_Post struct {
	Id                    int
	Name                  string
	Username              string
	Uploaded_avatar_id    int
	Avatar_template       string
	Created_at            string
	CreatedAtTime         time.Time
	Cooked                string
	Post_number           int
	Post_type             int
	Updated_at            string
	Like_count            int
	Reply_count           int
	Reply_to_post_number  int
	Quote_count           int
	Avg_time              int
	Incoming_link_count   int
	Reads                 int
	Score                 float64
	Yours                 bool
	Topic_id              int
	Topic_slug            string
	Display_username      string
	Primary_group_name    string
	Version               int
	Can_edit              bool
	Can_delete            bool
	Can_recover           bool
	User_title            string
	Raw                   string
	Actions_summary       []S_PostAction
	Moderator             bool
	Admin                 bool
	Staff                 bool
	User_id               int
	Hidden                bool
	Hidden_reason_id      int
	Trust_level           int
	Deleted_at            string
	User_deleted          bool
	Edit_reason           string
	Can_view_edit_history bool
	Wiki                  bool
}

func (s *S_Post) ParseTimes() {
	s.CreatedAtTime, _ = time.Parse(time.RFC3339, s.Created_at)
}


type S_PostStream struct {
	Posts  []S_Post
	Stream []int
}
type S_TopicParticipant struct {
	S_BasicUser
	Post_count int
}
type S_BasicTopic struct {
	Id                int
	Title             string
	Fancy_title       string
	Slug              string
	Posts_count       int
}
type S_ListedTopic struct {
	S_BasicTopic
	Reply_count           int
	Highest_post_number   int
	Image_url             string
	Created_at            string
	Bumped                bool
	Bumped_at             string
	Unseen                bool
	Last_read_post_number int
	Unread                int
	New_posts             int
	Pinned                bool
	Unpinned              bool
	Visible               bool
	Closed                bool
	Archived              bool
	Notification_level    int
	Bookmarked            bool
	Liked                 bool
}
type S_SuggestedTopic struct {
	S_ListedTopic
	Archetype   string
	Like_count  int
	Views       int
	Category_id int
}
type S_TopicListPoster struct {
	Extras      string
	Description string
	User_id     int
}
type S_TopicListItem struct {
	S_ListedTopic
	Views                int
	Like_count           int
	Has_summary          bool
	Archetype            string
	Last_poster_username string
	Category_id          int
	Pinned_globally      bool
	Posters              []S_TopicListPoster
}
type S_TopicLink struct {
	Url         string
	Title       string
	Fancy_title string
	Internal    bool
	Reflection  bool
	Clicks      int
	User_id     int
	Domain      string
}
type S_TopicDetails struct {
	Auto_close_at                 string
	Auto_close_hours              float64
	Auto_close_based_on_last_post bool
	Created_by                    S_BasicUser
	Last_posted                   S_BasicUser
	Participants                  []S_TopicParticipant
	Suggested_topics              []S_SuggestedTopic
	Links                         []S_TopicLink
	Notification_level            int
	Notifications_reason_id       int
	Can_edit                      bool
	Can_reply_as_new_topic        bool
	Can_flag_topic                bool
}
type S_TopicResponse struct {
	S_BasicTopic
	Created_at            string
	Views                 int
	Reply_count           int
	Image_url             string
	Highest_post_number   int
	Last_read_post_number int
	Participant_count     int
	Last_posted_at        string
	Like_count            int
	Visible               bool
	Closed                bool
	Archived              bool
	Has_summary           bool
	Archetype             string
	Category_id           int
	Word_count            int
	Deleted_at            string
}
type S_MessageBus struct {
	Global_Id  int
	Message_Id int
	Channel    string
	Data       map[string]interface{}
}

type ResponseUserSerializer struct {
	User_badges []S_UserBadge
	Badges      []S_Badge
	Badge_types []S_BadgeType
	Users       []S_BasicUser
	User        S_UserProfile
}
type ResponseLatestPosts struct {
	Latest_posts []S_Post
}
type ResponseTopic struct {
	Post_stream S_PostStream
	S_TopicResponse
	//	Draft ? what type
	Draft_key             string
	Draft_sequence        int
	Posted                bool
	Unpinned              bool
	Pinned_globally       bool
	Pinned                bool
	Pinned_at             string
	Details               S_TopicDetails
	Highest_post_number   int
	Last_read_post_number int
	Deleted_by            int
	Actions_summary       []S_PostAction
	Chunk_size            int
	Bookmarked            bool
}
type ResponseMessageBus []S_MessageBus
type ResponseNotifications []S_Notification
