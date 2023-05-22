package main

import "time"

type config struct {
	Token    string `fig:"token" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
	DSN      string `fig:"dsn" validate:"required"`
}

type channel struct {
	chatID     int64
	Feed       string
	LatestDate time.Time
}
