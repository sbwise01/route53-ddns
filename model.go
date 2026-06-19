package main

import (
	"fmt"
	"time"
)

type CustomError struct {
	ErrorCode int
	Err       error
}

func (err *CustomError) Error() string {
	return fmt.Sprintf("Error: %v, StatusCode: %d", err.Err, err.ErrorCode)
}

var (
	version             string        = "1.0.0-go1.23"
	defaultPollInterval time.Duration = 15 * time.Minute // Default poll interval
	gitrepo             string        = "https://github.com/sbwise01/route53-ddns"
	httpTimeout         time.Duration = 30 * time.Second
	defaultTTL          int64         = 300 // Default record TTL in seconds
)
