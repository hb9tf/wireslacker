package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/finfinack/wireslacker/data"
	"github.com/finfinack/wireslacker/processor"
	"github.com/finfinack/wireslacker/reader"
	"github.com/finfinack/wireslacker/resolver"
)

var (
	targets      = flag.String("targets", "", "coma separated paths or URLs to the log files")
	readInterval = flag.Duration("readInterval", 10*time.Second, "interval in which to read the provided logs")
	webHook      = flag.String("webhook", "", "webhook to use to post to slack")
	location     = flag.String("location", "Local", "location of the Wires-X server - see https://golang.org/pkg/time/#Location for details")
	verbose      = flag.Bool("v", false, "log more detailed messages")
	dry          = flag.Bool("dry", false, "do not post to slack channel if true")
)

// read uses the provided reader to read the log from target and sends the data.Log to the logChan.
func read(reader reader.Log, target string, verbose bool, logChan chan *data.Log) error {
	if verbose {
		log.Printf("V: Polling log %q", target)
	}
	evtLog, err := reader.Read()
	if err != nil {
		return err
	}
	logChan <- evtLog
	return nil
}

// readEvery reads the Wires-X log from the provided target every d and sends the
// parsed log to the provided logChan for further processing.
// Note that only non-recoverable errors should return. Retryable ones should log only.
func readEvery(d time.Duration, target string, verbose bool, logChan chan *data.Log, loc *time.Location) error {
	reader, err := reader.New(target, loc, verbose)
	if err != nil {
		return fmt.Errorf("unable to get reader: %v", err)
	}

	if err := read(reader, target, verbose, logChan); err != nil {
		log.Printf("Unable to poll log %q (temporarily?): %v", target, err) // we don't want to abort in this case and retry later
	}
	for _ = range time.Tick(d) {
		if err := read(reader, target, verbose, logChan); err != nil {
			log.Printf("Unable to poll log %q (temporarily?): %v", target, err)
			continue // we don't want to abort in this case and retry later
		}
	}
	return nil
}

func main() {
	flag.Parse()

	// Ensure necessary flags have been provided.
	if *webHook == "" {
		fmt.Println("provide a valid webhook URL for slack")
		os.Exit(1)
	}
	if *targets == "" {
		fmt.Println("provide at least one target")
		os.Exit(1)
	}

	loc, err := time.LoadLocation(*location)
	if err != nil {
		fmt.Printf("unable to parse provided location %q: %v\n", *location, err)
		os.Exit(1)
	}

	// Start auto-updating of active nodes cache.
	go resolver.AutoUpdate(*verbose)

	// Create log channel and start processing of incoming data.
	logChan := make(chan *data.Log)
	go processor.Run(logChan, processor.NewSlacker(*webHook, *dry), *verbose)

	// Start a reader for each target which has been provided.
	var wg sync.WaitGroup
	for _, target := range strings.Split(*targets, ",") {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			log.Printf("Start polling %q\n", target)
			if err := readEvery(*readInterval, target, *verbose, logChan, loc); err != nil {
				log.Printf("Unable to poll log %q (stopping): %v", target, err)
				return
			}
		}(target)
	}
	wg.Wait()
}
