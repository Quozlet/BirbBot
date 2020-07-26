package recurring

// Frequency at which the command should refresh
type Frequency int

const (
	// Daily refreshes every day (24 hours)
	Daily Frequency = iota
	// Hourly refreshes every hour (60 minutes)
	Hourly
	// HalfHourly refreshes every half-hour (30 minutes)
	HalfHourly
	// Minutely refreshes every minute (60 seconds)
	Minutely
)
