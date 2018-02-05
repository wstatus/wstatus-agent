package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/zmalik/wstatus-agent/pkg/utils"
	"github.com/zmalik/wstatus-agent/pkg/agent"
)

const (
	ENV_TOKEN_VAR = "WSTATUS_TOKEN"
)

func main() {
	log.Infoln("Initializing the wstatus agent...")
	worker := agent.NewWorker(utils.GetVariable(ENV_TOKEN_VAR))
	go worker.Run()
	select{}
}

