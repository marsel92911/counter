package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	MAX_GO_ROUTINES = 5
)

type Counter struct {
	n       int
	result  []int
	wg      *sync.WaitGroup
	mu      sync.Mutex
	limiter chan struct{}
	urls    []string
	log     *log.Logger
}

func NewCounter(maxProc int, urls []string) *Counter {
	ln := len(urls)
	wg := sync.WaitGroup{}
	wg.Add(ln)

	return &Counter{
		n:       ln,
		mu:      sync.Mutex{},
		result:  make([]int, ln),
		limiter: newLimiter(maxProc),
		wg:      &wg,
		urls:    urls,
		log:     log.Default(),
	}
}

func newLimiter(max int) chan struct{} {
	limiter := make(chan struct{}, max)
	for i := 0; i < max; i++ {
		limiter <- struct{}{}
	}

	return limiter
}

func (c *Counter) urlProcess(url string, ind int) {
	defer func() {
		c.limiter <- struct{}{}
		c.wg.Done()
	}()

	var count int

	resp, err := http.Get(url)
	if err != nil {
		c.log.Printf("Error sending GET response to: %s\n", url)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Printf("Error getting body from response. Url: %s\n", url)
		return
	}
	count = strings.Count(string(data), "Go")

	c.mu.Lock()
	c.result[ind] = count
	c.mu.Unlock()
}

func (c *Counter) Start() []int {
	for i, url := range c.urls {
		_ = <-c.limiter
		go c.urlProcess(url, i)
	}

	c.wg.Wait()
	return c.result
}

func main() {
	urls := []string{"https://golang.org", "https://go.dev", "https://ru.wikipedia.org/wiki/Go", "https://github.com/golang"}
	countWorker := NewCounter(MAX_GO_ROUTINES, urls)
	result := countWorker.Start()

	var total int
	for i := 0; i < len(result); i++ {
		total = total + result[i]
		fmt.Printf("Count for %s: %d\n", urls[i], result[i])
	}
	fmt.Printf("Total: %d\n", total)
}
