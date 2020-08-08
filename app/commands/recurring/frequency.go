package recurring

// Frequency at which the command should refresh
type Frequency int

const (
	// Daily refreshes every day (24 hours)
	Daily Frequency = iota
	// Hourly refreshes every hour (60 minutes)
	Hourly
	// Minutely refreshes every minute (60 seconds)
	Minutely
	// FiveMinutely refreshes every 5 minutes
	FiveMinutely
	// TenMinutely refreshes every 10 minutes
	TenMinutely
	// QuarterHourly refreshes every quarter hour (15 minutes)
	QuarterHourly
	// HalfHourly refreshes every half-hour (30 minutes)
	HalfHourly
	// QuarterToHourly refreshes every quarter to an hour (45 minutes)
	QuarterToHourly
)
