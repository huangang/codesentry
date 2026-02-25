package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/models"
	"github.com/huangang/codesentry/backend/pkg/response"
	"gorm.io/gorm"
)

// SearchHandler provides a global search across review logs, projects, and members.
type SearchHandler struct {
	db *gorm.DB
}

func NewSearchHandler(db *gorm.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

type SearchResult struct {
	Reviews  []ReviewSearchItem  `json:"reviews"`
	Projects []ProjectSearchItem `json:"projects"`
	Total    int                 `json:"total"`
}

type ReviewSearchItem struct {
	ID            uint     `json:"id"`
	ProjectID     uint     `json:"project_id"`
	ProjectName   string   `json:"project_name"`
	CommitHash    string   `json:"commit_hash"`
	CommitMessage string   `json:"commit_message"`
	Author        string   `json:"author"`
	Branch        string   `json:"branch"`
	Score         *float64 `json:"score"`
	ReviewStatus  string   `json:"review_status"`
	CreatedAt     string   `json:"created_at"`
}

type ProjectSearchItem struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Platform string `json:"platform"`
}

// Search performs a global search across reviews and projects.
func (h *SearchHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" || len(q) < 2 {
		response.BadRequest(c, "search query must be at least 2 characters")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 50 {
		limit = 20
	}

	result := SearchResult{}
	pattern := "%" + q + "%"

	// Search review logs
	var reviews []models.ReviewLog
	h.db.Model(&models.ReviewLog{}).
		Preload("Project").
		Where("commit_message LIKE ? OR author LIKE ? OR commit_hash LIKE ? OR branch LIKE ?",
			pattern, pattern, pattern, pattern).
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews)

	for _, r := range reviews {
		projectName := ""
		if r.Project != nil {
			projectName = r.Project.Name
		}
		result.Reviews = append(result.Reviews, ReviewSearchItem{
			ID:            r.ID,
			ProjectID:     r.ProjectID,
			ProjectName:   projectName,
			CommitHash:    r.CommitHash,
			CommitMessage: r.CommitMessage,
			Author:        r.Author,
			Branch:        r.Branch,
			Score:         r.Score,
			ReviewStatus:  r.ReviewStatus,
			CreatedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// Search projects
	var projects []models.Project
	h.db.Model(&models.Project{}).
		Where("name LIKE ? OR url LIKE ?", pattern, pattern).
		Limit(10).
		Find(&projects)

	for _, p := range projects {
		result.Projects = append(result.Projects, ProjectSearchItem{
			ID:       p.ID,
			Name:     p.Name,
			URL:      p.URL,
			Platform: p.Platform,
		})
	}

	result.Total = len(result.Reviews) + len(result.Projects)
	response.Success(c, result)
}
