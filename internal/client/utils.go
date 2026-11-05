package client

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func AskUserInput(msg string) string {
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-abort:
				fmt.Println("\nExiting...")
				os.Exit(0)
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	fmt.Printf("[CTRL+C to exit] %s: ", msg)

	var value string
	if _, err := fmt.Scanln(&value); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}

	value = strings.Replace(value, "\n", "", -1)

	return value
}
