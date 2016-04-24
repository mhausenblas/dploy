package dploy

import (
	log "github.com/Sirupsen/logrus"
)

func Init(location string) {
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", location)
}
