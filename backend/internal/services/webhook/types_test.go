package webhook

import (
	"encoding/json"
	"testing"
)

func TestGitLabPushEvent_Parse(t *testing.T) {
	jsonData := `{
		"object_kind": "push",
		"event_name": "push",
		"ref": "refs/heads/main",
		"checkout_sha": "abc123def456",
		"user_name": "John Doe",
		"user_email": "john@example.com",
		"project_id": 123,
		"project": {
			"name": "test-project",
			"web_url": "https://gitlab.com/test/project"
		},
		"commits": [
			{
				"id": "abc123",
				"message": "feat: add feature",
				"author": {
					"name": "John",
					"email": "john@example.com"
				}
			}
		],
		"total_commits_count": 1
	}`

	var event GitLabPushEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if event.ObjectKind != "push" {
		t.Errorf("ObjectKind = %q, expected %q", event.ObjectKind, "push")
	}
	if event.Ref != "refs/heads/main" {
		t.Errorf("Ref = %q, expected %q", event.Ref, "refs/heads/main")
	}
	if event.CheckoutSHA != "abc123def456" {
		t.Errorf("CheckoutSHA = %q, expected %q", event.CheckoutSHA, "abc123def456")
	}
	if event.UserName != "John Doe" {
		t.Errorf("UserName = %q, expected %q", event.UserName, "John Doe")
	}
	if event.ProjectID != 123 {
		t.Errorf("ProjectID = %d, expected %d", event.ProjectID, 123)
	}
	if len(event.Commits) != 1 {
		t.Errorf("Commits count = %d, expected %d", len(event.Commits), 1)
	}
	if event.Commits[0].ID != "abc123" {
		t.Errorf("Commit ID = %q, expected %q", event.Commits[0].ID, "abc123")
	}
}

func TestGitLabMREvent_Parse(t *testing.T) {
	jsonData := `{
		"object_kind": "merge_request",
		"user": {
			"name": "Jane Doe",
			"username": "jane",
			"email": "jane@example.com"
		},
		"project": {
			"id": 456,
			"name": "my-project",
			"web_url": "https://gitlab.com/org/my-project"
		},
		"object_attributes": {
			"iid": 42,
			"title": "Add new feature",
			"source_branch": "feature/new",
			"target_branch": "main",
			"state": "opened",
			"action": "open",
			"url": "https://gitlab.com/org/my-project/-/merge_requests/42"
		}
	}`

	var event GitLabMREvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if event.ObjectKind != "merge_request" {
		t.Errorf("ObjectKind = %q, expected %q", event.ObjectKind, "merge_request")
	}
	if event.User.Username != "jane" {
		t.Errorf("User.Username = %q, expected %q", event.User.Username, "jane")
	}
	if event.ObjectAttributes.IID != 42 {
		t.Errorf("IID = %d, expected %d", event.ObjectAttributes.IID, 42)
	}
	if event.ObjectAttributes.SourceBranch != "feature/new" {
		t.Errorf("SourceBranch = %q, expected %q", event.ObjectAttributes.SourceBranch, "feature/new")
	}
}

func TestGitHubPushEvent_Parse(t *testing.T) {
	jsonData := `{
		"ref": "refs/heads/main",
		"after": "def456abc789",
		"pusher": {
			"name": "developer",
			"email": "dev@example.com"
		},
		"sender": {
			"login": "developer",
			"avatar_url": "https://github.com/avatars/1"
		},
		"repository": {
			"id": 12345,
			"name": "my-repo",
			"full_name": "org/my-repo"
		},
		"commits": [
			{
				"id": "def456",
				"message": "fix: bug fix",
				"author": {
					"name": "Dev",
					"email": "dev@example.com"
				}
			}
		]
	}`

	var event GitHubPushEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if event.Ref != "refs/heads/main" {
		t.Errorf("Ref = %q, expected %q", event.Ref, "refs/heads/main")
	}
	if event.After != "def456abc789" {
		t.Errorf("After = %q, expected %q", event.After, "def456abc789")
	}
	if event.Pusher.Name != "developer" {
		t.Errorf("Pusher.Name = %q, expected %q", event.Pusher.Name, "developer")
	}
	if event.Repository.FullName != "org/my-repo" {
		t.Errorf("Repository.FullName = %q, expected %q", event.Repository.FullName, "org/my-repo")
	}
}

func TestGitHubPREvent_Parse(t *testing.T) {
	jsonData := `{
		"action": "opened",
		"number": 123,
		"pull_request": {
			"id": 999,
			"title": "New Feature",
			"state": "open",
			"head": {
				"ref": "feature-branch",
				"sha": "abc123"
			},
			"base": {
				"ref": "main"
			},
			"user": {
				"login": "contributor"
			},
			"html_url": "https://github.com/org/repo/pull/123"
		},
		"repository": {
			"id": 456,
			"name": "repo",
			"full_name": "org/repo"
		}
	}`

	var event GitHubPREvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if event.Action != "opened" {
		t.Errorf("Action = %q, expected %q", event.Action, "opened")
	}
	if event.Number != 123 {
		t.Errorf("Number = %d, expected %d", event.Number, 123)
	}
	if event.PullRequest.Head.SHA != "abc123" {
		t.Errorf("Head.SHA = %q, expected %q", event.PullRequest.Head.SHA, "abc123")
	}
	if event.PullRequest.Base.Ref != "main" {
		t.Errorf("Base.Ref = %q, expected %q", event.PullRequest.Base.Ref, "main")
	}
}

func TestBitbucketPushEvent_Parse(t *testing.T) {
	jsonData := `{
		"push": {
			"changes": [
				{
					"new": {
						"name": "main",
						"type": "branch",
						"target": {
							"hash": "abc123xyz",
							"message": "commit message",
							"author": {
								"raw": "User <user@example.com>"
							}
						}
					},
					"old": {
						"name": "main",
						"target": {
							"hash": "old123"
						}
					}
				}
			]
		},
		"repository": {
			"uuid": "{uuid-here}",
			"name": "my-repo",
			"full_name": "workspace/my-repo"
		},
		"actor": {
			"display_name": "Developer"
		}
	}`

	var event BitbucketPushEvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(event.Push.Changes) != 1 {
		t.Fatalf("Changes count = %d, expected 1", len(event.Push.Changes))
	}
	change := event.Push.Changes[0]
	if change.New.Name != "main" {
		t.Errorf("New.Name = %q, expected %q", change.New.Name, "main")
	}
	if change.New.Target.Hash != "abc123xyz" {
		t.Errorf("Target.Hash = %q, expected %q", change.New.Target.Hash, "abc123xyz")
	}
	if event.Repository.FullName != "workspace/my-repo" {
		t.Errorf("Repository.FullName = %q, expected %q", event.Repository.FullName, "workspace/my-repo")
	}
}

func TestBitbucketPREvent_Parse(t *testing.T) {
	jsonData := `{
		"pullrequest": {
			"id": 42,
			"title": "Feature PR",
			"description": "Description here",
			"state": "OPEN",
			"source": {
				"branch": {
					"name": "feature"
				},
				"commit": {
					"hash": "sourcehash"
				}
			},
			"destination": {
				"branch": {
					"name": "main"
				}
			},
			"author": {
				"display_name": "Author Name"
			}
		},
		"repository": {
			"full_name": "workspace/repo"
		},
		"actor": {
			"display_name": "Actor"
		}
	}`

	var event BitbucketPREvent
	err := json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if event.PullRequest.ID != 42 {
		t.Errorf("ID = %d, expected 42", event.PullRequest.ID)
	}
	if event.PullRequest.Title != "Feature PR" {
		t.Errorf("Title = %q, expected %q", event.PullRequest.Title, "Feature PR")
	}
	if event.PullRequest.Source.Branch.Name != "feature" {
		t.Errorf("Source.Branch.Name = %q, expected %q", event.PullRequest.Source.Branch.Name, "feature")
	}
	if event.PullRequest.Source.Commit.Hash != "sourcehash" {
		t.Errorf("Source.Commit.Hash = %q, expected %q", event.PullRequest.Source.Commit.Hash, "sourcehash")
	}
}

func TestSyncReviewRequest_Structure(t *testing.T) {
	req := SyncReviewRequest{
		ProjectURL: "https://github.com/org/repo",
		CommitSHA:  "abc123",
		Ref:        "refs/heads/main",
		Author:     "developer",
		Message:    "commit message",
		Diffs:      "diff content",
	}

	if req.ProjectURL != "https://github.com/org/repo" {
		t.Errorf("ProjectURL = %q", req.ProjectURL)
	}
	if req.CommitSHA != "abc123" {
		t.Errorf("CommitSHA = %q", req.CommitSHA)
	}
	if req.Ref != "refs/heads/main" {
		t.Errorf("Ref = %q, expected %q", req.Ref, "refs/heads/main")
	}
	if req.Author != "developer" {
		t.Errorf("Author = %q, expected %q", req.Author, "developer")
	}
	if req.Message != "commit message" {
		t.Errorf("Message = %q, expected %q", req.Message, "commit message")
	}
	if req.Diffs != "diff content" {
		t.Errorf("Diffs = %q, expected %q", req.Diffs, "diff content")
	}
}

func TestSyncReviewResponse_Structure(t *testing.T) {
	resp := SyncReviewResponse{
		Passed:      true,
		Score:       85.5,
		MinScore:    60,
		Message:     "Review passed",
		ReviewID:    123,
		FullContent: "Full review content",
	}

	if !resp.Passed {
		t.Error("Passed should be true")
	}
	if resp.Score != 85.5 {
		t.Errorf("Score = %f, expected 85.5", resp.Score)
	}
	if resp.MinScore != 60 {
		t.Errorf("MinScore = %f, expected 60", resp.MinScore)
	}
	if resp.Message != "Review passed" {
		t.Errorf("Message = %q, expected %q", resp.Message, "Review passed")
	}
	if resp.ReviewID != 123 {
		t.Errorf("ReviewID = %d, expected 123", resp.ReviewID)
	}
	if resp.FullContent != "Full review content" {
		t.Errorf("FullContent = %q, expected %q", resp.FullContent, "Full review content")
	}
}

func TestReviewScoreResponse_Structure(t *testing.T) {
	score := 75.0
	passed := true

	resp := ReviewScoreResponse{
		CommitSHA: "abc123",
		Status:    "completed",
		Score:     &score,
		MinScore:  60,
		Passed:    &passed,
		ReviewID:  456,
		Message:   "Review completed",
	}

	if resp.CommitSHA != "abc123" {
		t.Errorf("CommitSHA = %q", resp.CommitSHA)
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q", resp.Status)
	}
	if resp.Score == nil || *resp.Score != 75 {
		t.Error("Score should be 75")
	}
	if resp.MinScore != 60 {
		t.Errorf("MinScore = %f, expected 60", resp.MinScore)
	}
	if resp.Passed == nil || *resp.Passed != true {
		t.Error("Passed should be true")
	}
	if resp.ReviewID != 456 {
		t.Errorf("ReviewID = %d, expected 456", resp.ReviewID)
	}
	if resp.Message != "Review completed" {
		t.Errorf("Message = %q, expected %q", resp.Message, "Review completed")
	}
}
