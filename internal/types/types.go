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
