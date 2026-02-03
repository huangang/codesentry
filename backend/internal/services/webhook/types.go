package webhook

// GitLabPushEvent represents a GitLab push webhook event
type GitLabPushEvent struct {
	ObjectKind  string `json:"object_kind"`
	EventName   string `json:"event_name"`
	Ref         string `json:"ref"`
	CheckoutSHA string `json:"checkout_sha"`
	UserName    string `json:"user_name"`
	UserEmail   string `json:"user_email"`
	UserAvatar  string `json:"user_avatar"`
	ProjectID   int    `json:"project_id"`
	Project     struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		WebURL    string `json:"web_url"`
		Namespace string `json:"namespace"`
	} `json:"project"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
	TotalCommitsCount int `json:"total_commits_count"`
}

// GitLabMREvent represents a GitLab merge request webhook event
type GitLabMREvent struct {
	ObjectKind string `json:"object_kind"`
	User       struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
	Project struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		WebURL    string `json:"web_url"`
		Namespace string `json:"namespace"`
	} `json:"project"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		State        string `json:"state"`
		Action       string `json:"action"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
}

// GitHubPushEvent represents a GitHub push webhook event
type GitHubPushEvent struct {
	Ref    string `json:"ref"`
	After  string `json:"after"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Sender struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"sender"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		URL      string `json:"url"`
	} `json:"repository"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
		Author    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

// GitHubPREvent represents a GitHub pull request webhook event
type GitHubPREvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Body  string `json:"body"`
		State string `json:"state"`
		Head  struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		User struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
			HTMLURL   string `json:"html_url"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
	} `json:"pull_request"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// BitbucketPushEvent represents a Bitbucket push webhook event
type BitbucketPushEvent struct {
	Push struct {
		Changes []struct {
			New struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Target struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
					Date    string `json:"date"`
					Author  struct {
						Raw  string `json:"raw"`
						User struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
							AccountID   string `json:"account_id"`
							Nickname    string `json:"nickname"`
							Links       struct {
								Avatar struct {
									Href string `json:"href"`
								} `json:"avatar"`
								HTML struct {
									Href string `json:"href"`
								} `json:"html"`
							} `json:"links"`
						} `json:"user"`
					} `json:"author"`
					Links struct {
						HTML struct {
							Href string `json:"href"`
						} `json:"html"`
					} `json:"links"`
				} `json:"target"`
			} `json:"new"`
			Old struct {
				Name   string `json:"name"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"old"`
			Commits []struct {
				Hash    string `json:"hash"`
				Message string `json:"message"`
				Author  struct {
					Raw  string `json:"raw"`
					User struct {
						DisplayName string `json:"display_name"`
					} `json:"user"`
				} `json:"author"`
				Links struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"commits"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Links    struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		AccountID   string `json:"account_id"`
		Nickname    string `json:"nickname"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"actor"`
}

// BitbucketPREvent represents a Bitbucket pull request webhook event
type BitbucketPREvent struct {
	PullRequest struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		Source      struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
		Author struct {
			DisplayName string `json:"display_name"`
			UUID        string `json:"uuid"`
			AccountID   string `json:"account_id"`
			Nickname    string `json:"nickname"`
			Links       struct {
				Avatar struct {
					Href string `json:"href"`
				} `json:"avatar"`
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
		} `json:"author"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"pullrequest"`
	Repository struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Links    struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
		Links       struct {
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"actor"`
}

// ReviewScoreResponse represents the response for commit score queries
type ReviewScoreResponse struct {
	CommitSHA string   `json:"commit_sha"`
	Status    string   `json:"status"`
	Score     *float64 `json:"score,omitempty"`
	MinScore  float64  `json:"min_score,omitempty"`
	Passed    *bool    `json:"passed,omitempty"`
	ReviewID  uint     `json:"review_id"`
	Message   string   `json:"message"`
}

// SyncReviewRequest represents a synchronous review request
type SyncReviewRequest struct {
	ProjectURL string
	CommitSHA  string
	Ref        string
	Author     string
	Message    string
	Diffs      string
}

// SyncReviewResponse represents a synchronous review response
type SyncReviewResponse struct {
	Passed      bool    `json:"passed"`
	Score       float64 `json:"score"`
	MinScore    float64 `json:"min_score"`
	Message     string  `json:"message"`
	ReviewID    uint    `json:"review_id,omitempty"`
	FullContent string  `json:"full_content,omitempty"`
}
