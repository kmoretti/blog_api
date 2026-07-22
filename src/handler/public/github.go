package public

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var githubPathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

var githubHTTPClient = &http.Client{Timeout: 10 * time.Second}

func GitHubRepository(c *gin.Context) {
	owner := c.Param("owner")
	repo := strings.TrimSuffix(c.Param("repo"), ".git")
	if !githubPathSegmentPattern.MatchString(owner) || !githubPathSegmentPattern.MatchString(repo) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid GitHub repository path"})
		return
	}

	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, "https://api.github.com/repos/"+owner+"/"+repo, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create GitHub request"})
		return
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := githubHTTPClient.Do(request)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "GitHub request failed"})
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		c.Status(response.StatusCode)
		return
	}

	var payload json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid GitHub response"})
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}
