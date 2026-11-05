package client

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
)

const SampleSize int = 128

type PerformanceCheck struct {
	*sync.Mutex
	*sync.WaitGroup
	Samples *[]bool
}

func (check *PerformanceCheck) hitServer(url string) {
	defer check.Done()

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		check.Lock()
		*check.Samples = append(*check.Samples, true)
		check.Unlock()
	}
}

// Test the server hardness. The program will spawn processes that will all hit the server ~the same millisecond
func CheckPerformance(ctx context.Context, url string, sampleSize int) error {
	if sampleSize <= 0 {
		sampleSize = SampleSize
	}

	samples := make([]bool, 0, sampleSize)

	if url == "" {
		url = "http://localhost:8000/health"
	}

	check := &PerformanceCheck{
		Mutex:     &sync.Mutex{},
		WaitGroup: &sync.WaitGroup{},
		Samples:   &samples,
	}

	for range sampleSize {
		check.Add(1)
		go check.hitServer(url)
	}
	check.Wait()

	successful := len(samples)

	percent := int((float32(successful) / float32(sampleSize)) * 100)
	log.Printf("%d of %d samples returned successfully. (%d%%)", successful, sampleSize, percent)

	return nil
}
