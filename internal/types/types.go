package types

// DORAMetrics represents DORA metrics data
type DORAMetrics struct {
	DeploymentFrequency  *DeploymentFrequency  `json:"deploymentFrequency"`
	LeadTimeForChanges   *LeadTimeForChanges   `json:"leadTimeForChanges"`
	TimeToRestoreService *TimeToRestoreService `json:"timeToRestoreService"`
	ChangeFailureRate    *ChangeFailureRate    `json:"changeFailureRate"`
}

type DeploymentFrequency struct {
	DeploymentsPerDay float64 `json:"deploymentsPerDay"`
	Rating            string  `json:"rating"`
	Trend             string  `json:"trend"`
}

type LeadTimeForChanges struct {
	AverageHours float64 `json:"averageHours"`
	Rating       string  `json:"rating"`
	Trend        string  `json:"trend"`
}

type TimeToRestoreService struct {
	AverageHours float64 `json:"averageHours"`
	Rating       string  `json:"rating"`
	Trend        string  `json:"trend"`
}

type ChangeFailureRate struct {
	Percentage float64 `json:"percentage"`
	Rating     string  `json:"rating"`
	Trend      string  `json:"trend"`
}

// HealthStatus represents system health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Services  map[string]interface{} `json:"services,omitempty"`
}

// Repository represents a GitHub repository
type Repository struct {
	ID       int    `json:"id"`
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	FullName string `json:"full_name,omitempty"`
}

// RateLimitStatus represents GitHub API rate limit status
type RateLimitStatus struct {
	Remaining   int     `json:"remaining"`
	Limit       int     `json:"limit"`
	ResetIn     int     `json:"resetIn"`
	ResetAt     string  `json:"resetAt"`
	PercentUsed float64 `json:"percentUsed"`
}

// ============================================================
// MULTI-PROVIDER GIT TYPES
// ============================================================

// GitProvider represents supported Git providers
type GitProvider string

const (
	ProviderGitHub      GitProvider = "github"
	ProviderGitLab      GitProvider = "gitlab"
	ProviderBitbucket   GitProvider = "bitbucket"
	ProviderAzureDevOps GitProvider = "azure_devops"
)

// ProviderInfo contains information about a git provider
type ProviderInfo struct {
	ID           GitProvider `json:"id"`
	DisplayName  string      `json:"displayName"`
	Icon         string      `json:"icon"`
	Color        string      `json:"color"`
	IsConnected  bool        `json:"isConnected"`
	Capabilities []string    `json:"capabilities"`
}

// ProviderConnection represents a connection to a git provider
type ProviderConnection struct {
	ID             int         `json:"id"`
	Provider       GitProvider `json:"provider"`
	DisplayName    string      `json:"displayName"`
	BaseURL        string      `json:"baseUrl,omitempty"`
	ConnectionType string      `json:"connectionType"` // "organization" or "user"
	IsActive       bool        `json:"isActive"`
	CreatedAt      string      `json:"createdAt"`
	LastUsedAt     string      `json:"lastUsedAt,omitempty"`
}

// UnifiedRepository represents a repository from any provider
type UnifiedRepository struct {
	ID              string      `json:"id"`
	Provider        GitProvider `json:"provider"`
	ProviderRepoID  string      `json:"providerRepoId"`
	FullName        string      `json:"fullName"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	URL             string      `json:"url"`
	CloneURL        string      `json:"cloneUrl,omitempty"`
	DefaultBranch   string      `json:"defaultBranch"`
	IsPrivate       bool        `json:"isPrivate"`
	IsArchived      bool        `json:"isArchived"`
	Language        string      `json:"language,omitempty"`
	StarCount       int         `json:"starCount,omitempty"`
	ForkCount       int         `json:"forkCount,omitempty"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
	Owner           RepoOwner   `json:"owner"`
}

// RepoOwner represents the owner of a repository
type RepoOwner struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// UnifiedPullRequest represents a PR/MR from any provider
type UnifiedPullRequest struct {
	ID              string      `json:"id"`
	Provider        GitProvider `json:"provider"`
	ProviderPRID    string      `json:"providerPrId"`
	Number          int         `json:"number"`
	Title           string      `json:"title"`
	Description     string      `json:"description,omitempty"`
	State           string      `json:"state"` // open, merged, closed
	Author          string      `json:"author"`
	AuthorAvatarURL string      `json:"authorAvatarUrl,omitempty"`
	CreatedAt       string      `json:"createdAt"`
	UpdatedAt       string      `json:"updatedAt"`
	MergedAt        string      `json:"mergedAt,omitempty"`
	ClosedAt        string      `json:"closedAt,omitempty"`
	URL             string      `json:"url"`
	RepoFullName    string      `json:"repoFullName"`
	SourceBranch    string      `json:"sourceBranch"`
	TargetBranch    string      `json:"targetBranch"`
	IsDraft         bool        `json:"isDraft"`
	Reviewers       []string    `json:"reviewers,omitempty"`
	Labels          []string    `json:"labels,omitempty"`
	Additions       int         `json:"additions,omitempty"`
	Deletions       int         `json:"deletions,omitempty"`
	ChangedFiles    int         `json:"changedFiles,omitempty"`
}

// UnifiedCommit represents a commit from any provider
type UnifiedCommit struct {
	ID           string       `json:"id"`
	Provider     GitProvider  `json:"provider"`
	SHA          string       `json:"sha"`
	ShortSHA     string       `json:"shortSha"`
	Message      string       `json:"message"`
	Author       CommitAuthor `json:"author"`
	CreatedAt    string       `json:"createdAt"`
	URL          string       `json:"url"`
	RepoFullName string       `json:"repoFullName"`
	Additions    int          `json:"additions,omitempty"`
	Deletions    int          `json:"deletions,omitempty"`
}

// CommitAuthor represents commit author information
type CommitAuthor struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// UnifiedPipeline represents a CI/CD pipeline from any provider
type UnifiedPipeline struct {
	ID                 string      `json:"id"`
	Provider           GitProvider `json:"provider"`
	ProviderPipelineID string      `json:"providerPipelineId"`
	Name               string      `json:"name,omitempty"`
	Status             string      `json:"status"` // success, failure, pending, running, cancelled, skipped
	Conclusion         string      `json:"conclusion,omitempty"`
	CreatedAt          string      `json:"createdAt"`
	StartedAt          string      `json:"startedAt,omitempty"`
	FinishedAt         string      `json:"finishedAt,omitempty"`
	Duration           int         `json:"duration,omitempty"` // seconds
	URL                string      `json:"url,omitempty"`
	CommitSHA          string      `json:"commitSha"`
	Branch             string      `json:"branch,omitempty"`
	RepoFullName       string      `json:"repoFullName"`
	Event              string      `json:"event,omitempty"`
}

// ProviderSummary provides an overview of a connected provider
type ProviderSummary struct {
	Provider         GitProvider    `json:"provider"`
	RepositoryCount  int            `json:"repositoryCount"`
	OpenPullRequests int            `json:"openPullRequests"`
	RecentCommits    int            `json:"recentCommits"`
	PipelineStatus   PipelineStats  `json:"pipelineStatus"`
	IsConnected      bool           `json:"isConnected"`
	LastSyncAt       string         `json:"lastSyncAt,omitempty"`
	Error            string         `json:"error,omitempty"`
}

// PipelineStats contains pipeline statistics
type PipelineStats struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
	Running int `json:"running"`
}

// AggregatedResult contains results aggregated across providers
type AggregatedResult struct {
	Items      interface{}            `json:"items"`
	TotalCount int                    `json:"totalCount"`
	ByProvider map[GitProvider]int    `json:"byProvider"`
	Errors     map[GitProvider]string `json:"errors,omitempty"`
}

// ConnectProviderRequest represents a request to connect a provider
type ConnectProviderRequest struct {
	Provider       GitProvider `json:"provider"`
	ConnectionType string      `json:"connectionType"` // "organization" or "user"
	BaseURL        string      `json:"baseUrl,omitempty"`
}

// OAuthCallbackResponse represents the OAuth callback response
type OAuthCallbackResponse struct {
	Success      bool   `json:"success"`
	Provider     string `json:"provider"`
	ConnectionID int    `json:"connectionId,omitempty"`
	Message      string `json:"message,omitempty"`
	RedirectURL  string `json:"redirectUrl,omitempty"`
}
