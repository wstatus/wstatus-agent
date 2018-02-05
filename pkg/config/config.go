package config

import (
	"github.com/zmalik/wstatus-agent/pkg/utils"
	"time"
)

const (
	ENV_ENDPOINT = "WSTATUS_ENDPOINT"
	ENV_DEFAULT_POLL_TIME = "WSTATUS_POLL"
)

func GetEndpoint() string{
	return utils.GetStringWithDefault(ENV_ENDPOINT, "http://localhost:8008/api/")
}


func GetDefaultPolling() time.Duration{
	return utils.GetDurationWithDefault(ENV_DEFAULT_POLL_TIME, 30*time.Second)
}


