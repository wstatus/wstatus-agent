package utils

import (
	"github.com/spf13/viper"
	"time"
)

func GetVariable(variable string) string {
	return viper.GetString(variable)
}

func GetStringWithDefault(variable, def string) string {
	if len(GetVariable(variable)) == 0 {
		return def
	}
	return GetVariable(variable)
}

func GetDurationWithDefault(variable string, def time.Duration) time.Duration {
	if viper.GetDuration(variable) == 0 {
		return def
	}
	return viper.GetDuration(variable)
}

func init() {
	viper.AutomaticEnv()
}
