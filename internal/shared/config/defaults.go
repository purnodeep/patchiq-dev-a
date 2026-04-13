package config

import "time"

// DefaultScanConfig returns Level 0 system defaults for the scan module.
func DefaultScanConfig() ScanConfig {
	schedule := "0 2 * * *"
	maxConcurrent := 5
	return ScanConfig{
		Schedule:      &schedule,
		MaxConcurrent: &maxConcurrent,
	}
}

// DefaultDeployConfig returns Level 0 system defaults for the deploy module.
func DefaultDeployConfig() DeployConfig {
	autoReboot := true
	rebootDelay := Duration{Duration: 5 * time.Minute}
	maxConcurrent := 10
	waveStrategy := "sequential"
	notifyUser := true
	bwLimit := "50mbps"
	return DeployConfig{
		AutoReboot:     &autoReboot,
		RebootDelay:    &rebootDelay,
		MaxConcurrent:  &maxConcurrent,
		WaveStrategy:   &waveStrategy,
		NotifyUser:     &notifyUser,
		BandwidthLimit: &bwLimit,
		MaintenanceWindow: &TimeWindow{
			Day:   "sunday",
			Start: "02:00",
			End:   "06:00",
		},
	}
}

// DefaultNotificationConfig returns Level 0 system defaults for notifications.
func DefaultNotificationConfig() NotificationConfig {
	emailEnabled := true
	slackEnabled := false
	return NotificationConfig{
		EmailEnabled: &emailEnabled,
		SlackEnabled: &slackEnabled,
	}
}

// DefaultDiscoveryConfig returns Level 0 system defaults for the discovery module.
func DefaultDiscoveryConfig() DiscoveryConfig {
	schedule := "0 * * * *"
	syncIntervalMins := 60
	httpTimeout := 120
	maxRetries := 3
	return DiscoveryConfig{
		Schedule:         &schedule,
		SyncIntervalMins: &syncIntervalMins,
		HTTPTimeout:      &httpTimeout,
		MaxRetries:       &maxRetries,
		Repositories: []RepositoryConfig{
			{
				Name:     "ubuntu-22.04-security",
				Type:     "apt",
				URL:      "http://archive.ubuntu.com/ubuntu/dists/jammy-security/main/binary-amd64/Packages.gz",
				OsFamily: "debian",
				OsDistro: "ubuntu-22.04",
				Enabled:  true,
			},
		},
	}
}

// DefaultCVEConfig returns Level 0 system defaults for the CVE module.
func DefaultCVEConfig() CVEConfig {
	schedule := "0 2 * * *"
	syncIntervalMins := 1440 // 24 hours
	httpTimeout := 60
	maxRetries := 3
	return CVEConfig{
		Schedule:         &schedule,
		SyncIntervalMins: &syncIntervalMins,
		HTTPTimeout:      &httpTimeout,
		MaxRetries:       &maxRetries,
	}
}

// DefaultAgentConfig returns Level 0 system defaults for agent behavior.
func DefaultAgentConfig() AgentConfig {
	heartbeat := Duration{Duration: 1 * time.Minute}
	logLevel := "info"
	selfUpdate := true
	return AgentConfig{
		HeartbeatInterval: &heartbeat,
		LogLevel:          &logLevel,
		SelfUpdateEnabled: &selfUpdate,
	}
}
