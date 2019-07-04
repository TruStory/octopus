package main

// SystemMetrics represents metrics for the platform.
type SystemMetrics struct {
	Users map[string]*UserMetricsV2 `json:"users"`
}

// MetricsV2 defines the numbers that are tracked
type MetricsV2 struct {
	// StakeBased Metrics
	TotalAmountStaked  Coin `json:"total_amount_staked"`
	StakeEarned        Coin `json:"stake_earned"`
	StakeLost          Coin `json:"stake_lost"`
	TotalAmountAtStake Coin `json:"total_amount_at_stake"`
	AvailableStake     Coin `json:"available_stake"`
}

// UserMetricsV2 a summary of different metrics per user
type UserMetricsV2 struct {
	// For each community
	CommunityMetrics map[string]*CommunityMetrics `json:"community_metrics"`
}

// CommunityMetrics summary of metrics by community
type CommunityMetrics struct {
	CommunityID string     `json:"community_id"`
	Metrics     *MetricsV2 `json:"metrics"`
}
