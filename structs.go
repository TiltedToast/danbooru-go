package main

type inputOptions struct {
	tags         []string
	outputDir    string
	sensitive    bool
	questionable bool
	explicit     bool
	general      bool
}

type Post struct {
	ID      int    `json:"id"`
	Score   int    `json:"score"`
	Rating  string `json:"rating"`
	FileExt string `json:"file_ext"`
	FileURL string `json:"file_url"`
}

type User struct {
	LastLoggedInAt string `json:"last_logged_in_at"`
	ID int `json:"id"`
	Name string `json:"name"`
	Level int `json:"level"`
	InviterID interface{} `json:"inviter_id"`
	CreatedAt string `json:"created_at"`
	LastForumReadAt string `json:"last_forum_read_at"`
	CommentThreshold int `json:"comment_threshold"`
	UpdatedAt string `json:"updated_at"`
	DefaultImageSize string `json:"default_image_size"`
	FavoriteTags interface{} `json:"favorite_tags"`
	BlacklistedTags string `json:"blacklisted_tags"`
	TimeZone string `json:"time_zone"`
	PostUpdateCount int `json:"post_update_count"`
	NoteUpdateCount int `json:"note_update_count"`
	FavoriteCount int `json:"favorite_count"`
	PostUploadCount int `json:"post_upload_count"`
	PerPage int `json:"per_page"`
	CustomStyle interface{} `json:"custom_style"`
	Theme string `json:"theme"`
	IsDeleted bool `json:"is_deleted"`
	LevelString string `json:"level_string"`
	IsBanned bool `json:"is_banned"`
	ReceiveEmailNotifications bool `json:"receive_email_notifications"`
	NewPostNavigationLayout bool `json:"new_post_navigation_layout"`
	EnablePrivateFavorites bool `json:"enable_private_favorites"`
	ShowDeletedChildren bool `json:"show_deleted_children"`
	DisableCategorizedSavedSearches bool `json:"disable_categorized_saved_searches"`
	DisableTaggedFilenames bool `json:"disable_tagged_filenames"`
	DisableMobileGestures bool `json:"disable_mobile_gestures"`
	EnableSafeMode bool `json:"enable_safe_mode"`
	EnableDesktopMode bool `json:"enable_desktop_mode"`
	DisablePostTooltips bool `json:"disable_post_tooltips"`
	RequiresVerification bool `json:"requires_verification"`
	IsVerified bool `json:"is_verified"`
	ShowDeletedPosts bool `json:"show_deleted_posts"`
	StatementTimeout int `json:"statement_timeout"`
	FavoriteGroupLimit int `json:"favorite_group_limit"`
	TagQueryLimit int `json:"tag_query_limit"`
	MaxSavedSearches int `json:"max_saved_searches"`
	WikiPageVersionCount int `json:"wiki_page_version_count"`
	ArtistVersionCount int `json:"artist_version_count"`
	ArtistCommentaryVersionCount int `json:"artist_commentary_version_count"`
	PoolVersionCount int `json:"pool_version_count"`
	ForumPostCount int `json:"forum_post_count"`
	CommentCount int `json:"comment_count"`
	FavoriteGroupCount int `json:"favorite_group_count"`
	AppealCount int `json:"appeal_count"`
	FlagCount int `json:"flag_count"`
	PositiveFeedbackCount int `json:"positive_feedback_count"`
	NeutralFeedbackCount int `json:"neutral_feedback_count"`
	NegativeFeedbackCount int `json:"negative_feedback_count"`
}