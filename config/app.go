package config

const (
	TriggerModifyFilename = ".changed"
)

// redis stream
const (
	Stream = "jobs"
	Group  = "workers"

	MaxJobRetryLimit = 10
)
