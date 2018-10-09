package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/finfinack/wireslack/data"
	"github.com/finfinack/wireslack/processor"
	"github.com/finfinack/wireslack/reader"
)

var (
	targets      = flag.String("targets", "", "coma separated paths or URLs to the log files")
	readInterval = flag.Duration("readInterval", 10*time.Second, "interval in which to read the provided logs")
	webHook      = flag.String("webhook", "", "webhook to use to post to slack")
	dry          = flag.Bool("dry", false, "do not post to slack channel if true")
)

// readEvery reads the Wires-X log from the provided target every d and sends the
// parsed log to the provided logChan for further processing.
func readEvery(d time.Duration, target string, logChan chan *data.Log) error {
	reader, err := reader.New(target)
	if err != nil {
		return fmt.Errorf("unable to get reader: %v", err)
	}

	for now := range time.Tick(d) {
		log.Printf("Polling log %q: %v\n", target, now)
		log, err := reader.Read()
		if err != nil {
			return err
		}
		logChan <- log
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

	// Create log channel and start processing of incoming data.
	logChan := make(chan *data.Log)
	go processor.Run(logChan, processor.NewSlacker(*webHook, *dry))

	// Start a reader for each target which has been provided.
	var wg sync.WaitGroup
	for _, target := range strings.Split(*targets, ",") {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			log.Printf("Start polling %q\n", target)
			if err := readEvery(*readInterval, target, logChan); err != nil {
				fmt.Println(err)
				return
			}
		}(target)
	}
	wg.Wait()
}
