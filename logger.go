package main

import (
	"fmt"
	"log"
	"os"
)

const (
	ErrorLog       string = "ERROR"
	InformationLog string = "INFO"
	WarningLog     string = "WARN"
)

// DDNSLogger writes a structured, timestamped line to stdout. The fqdn argument
// is built from the host and domain so log lines are easy to correlate with the
// record being managed. ERROR logs are fatal to match the original behavior of
// failing fast on unrecoverable startup errors.
func DDNSLogger(logType, fqdn, message string) {
	var (
		StdoutInfoLogger    *log.Logger
		StdoutWarningLogger *log.Logger
		StdoutErrorLogger   *log.Logger
	)

	StdoutInfoLogger = log.New(os.Stdout, "INFO ", log.Ldate|log.Ltime)
	StdoutWarningLogger = log.New(os.Stdout, "WARNING ", log.Ldate|log.Ltime)
	StdoutErrorLogger = log.New(os.Stdout, "ERROR ", log.Ldate|log.Ltime)

	switch logType {
	case InformationLog:
		StdoutInfoLogger.Println(fqdn, message)
	case WarningLog:
		StdoutWarningLogger.Println(fqdn, message)
	case ErrorLog:
		StdoutErrorLogger.Fatalln(fqdn, message)
	default:
		fmt.Println(fqdn, message)
	}
}
