package client

// The shapes below mirror what the Nook API returns. They are deliberately
// partial: the CLI decodes the fields it displays and ignores the rest, so a
// new field on the server is not a breaking change here.

// Provider is one configured source.
type Provider struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Tags    []string `json:"tags"`
	URL     string   `json:"url,omitempty"`
	Path    string   `json:"path,omitempty"`
	FeedURL string   `json:"feedUrl,omitempty"`
	Host    string   `json:"host,omitempty"`
}

// Endpoint is the address a provider watches or answers on, whichever applies.
func (p Provider) Endpoint() string {
	switch {
	case p.URL != "":
		return p.URL
	case p.Path != "":
		return p.Path
	case p.FeedURL != "":
		return p.FeedURL
	case p.Host != "":
		return p.Host
	}
	return ""
}

// Target is one configured delivery channel.
type Target struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	ProviderIDs  []string `json:"providerIds"`
	ProviderTags []string `json:"providerTags"`
}

// Silent reports whether the routing rules can ever select this target. Nook's
// routing is opt-in with no fallthrough, so a target naming no provider and no
// tag receives nothing — the most common reason one looks broken.
func (t Target) Silent() bool {
	return len(t.ProviderIDs) == 0 && len(t.ProviderTags) == 0
}

// Settings is the configuration document.
type Settings struct {
	Delivery struct {
		Items []Target `json:"items"`
	} `json:"delivery"`
	Providers struct {
		Port  int        `json:"port"`
		Items []Provider `json:"items"`
	} `json:"providers"`
	Timeout int `json:"timeout"`
}

// Runtime is what the instance reports about itself.
type Runtime struct {
	ServerPort             int    `json:"serverPort"`
	SettingsPath           string `json:"settingsPath"`
	UsingPersistedSettings bool   `json:"usingPersistedSettings"`
	DashboardBuilt         bool   `json:"dashboardBuilt"`
	LastReloadedAt         string `json:"lastReloadedAt"`
}

// SettingsResponse is what GET and PUT /api/settings return.
type SettingsResponse struct {
	Settings         Settings `json:"settings"`
	Runtime          Runtime  `json:"runtime"`
	MatrixConfigured bool     `json:"matrixConfigured"`
}

// Attempt is one delivery of one event to one target.
type Attempt struct {
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName"`
	TargetType string `json:"targetType"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"durationMs"`
}

// Event is one entry in the activity log.
type Event struct {
	ID           string    `json:"id"`
	Timestamp    string    `json:"timestamp"`
	Source       string    `json:"source"`
	ProviderID   *string   `json:"providerId"`
	ProviderName string    `json:"providerName"`
	ProviderType string    `json:"providerType"`
	Event        string    `json:"event"`
	Message      *string   `json:"message"`
	Deliveries   []Attempt `json:"deliveries"`
	Envelope     any       `json:"envelope,omitempty"`
}

// Delivered and Failed count the attempts by outcome.
func (e Event) Delivered() int { return e.count("success") }
func (e Event) Failed() int    { return e.count("error") }

func (e Event) count(status string) int {
	total := 0
	for _, attempt := range e.Deliveries {
		if attempt.Status == status {
			total++
		}
	}
	return total
}

// Stats summarize a slice of the activity log.
type Stats struct {
	TotalEvents          int     `json:"totalEvents"`
	LastEventAt          *string `json:"lastEventAt"`
	DeliverySuccessCount int     `json:"deliverySuccessCount"`
	DeliveryErrorCount   int     `json:"deliveryErrorCount"`
}

// EventPage is one page of the activity log plus the stats for the whole
// filtered set — not just the page, which is what makes the counters stable.
type EventPage struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
	Stats  Stats   `json:"stats"`
}

// PoolApp is one app connected to the Nook Pool.
type PoolApp struct {
	AppID    string   `json:"app_id"`
	App      string   `json:"app"`
	Channels []string `json:"channels"`
}

// PoolStats is the live state of the pool.
type PoolStats struct {
	Epoch       string    `json:"epoch"`
	Connections int       `json:"connections"`
	Apps        []PoolApp `json:"apps"`
}

// Session is who the instance thinks the caller is.
type Session struct {
	Authenticated    bool   `json:"authenticated"`
	PasswordRequired bool   `json:"passwordRequired"`
	Username         string `json:"username"`
}

// EventQuery filters the activity log.
type EventQuery struct {
	Limit      int
	Offset     int
	Search     string
	Source     string
	ProviderID string
	TargetID   string
}
