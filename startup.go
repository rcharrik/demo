package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

type RateLimiter struct {
	KV           *api.KV
	Key          string
	Tokens       int
	LastRefilled time.Time

	mu sync.Mutex
}

func NewRateLimiter(kv *api.KV, key string) (*RateLimiter, error) {
	rl := &RateLimiter{
		KV:  kv,
		Key: key,
	}
	fmt.Println("----------a")

	// Fetch the rate limiter configuration from Consul on initialization
	if err := rl.fetchRateLimiterConfig(); err != nil {
		fmt.Println("----------b")
		if err := rl.storeRateLimiterInConsul(); err != nil {
			return nil, err
		}
		fmt.Println("----------c")
	}
	fmt.Println("----------d")
	return rl, nil
}

func (rl *RateLimiter) fetchRateLimiterConfig() error {
	rl.mu.Lock() // Lock before accessing shared state
	defer rl.mu.Unlock()
	fmt.Println("----------x")

	pair, _, err := rl.KV.Get(rl.Key, nil)
	fmt.Println("----------y")

	if err != nil {
		return err
	}
	fmt.Println("----------z")

	if pair == nil || pair.Value == nil {
		return fmt.Errorf("Rate limiter configuration not found in Consul")
	}
	fmt.Println("----------t")

	return rl.parseRateLimiterConfig(pair.Value)
}

func (rl *RateLimiter) parseRateLimiterConfig(data []byte) error {
	rl.mu.Lock() // Lock before accessing shared state
	defer rl.mu.Unlock()

	// Parse the rate limiter configuration from Consul
	// Modify this function to parse the specific fields you store in Consul
	// In this example, we assume "Tokens" and "LastRefilled" are stored
	// and we'll parse them as JSON
	var config struct {
		Tokens       int       `json:"Tokens"`
		LastRefilled time.Time `json:"LastRefilled"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	rl.Tokens = config.Tokens
	rl.LastRefilled = config.LastRefilled

	return nil
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Fetch the rate limiter configuration from Consul before checking
	if err := rl.fetchRateLimiterConfig(); err != nil {
		log.Printf("Error fetching rate limiter configuration: %v", err)
		return false // Rate limit exceeded by default
	}

	now := time.Now()
	elapsed := now.Sub(rl.LastRefilled)

	// Calculate how many tokens to add based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.Tokens))
	if tokensToAdd > 0 {
		rl.Tokens = min(rl.Tokens+tokensToAdd, rl.Tokens)
		rl.LastRefilled = now

		// Update the rate limiter state in Consul
		if err := rl.storeRateLimiterInConsul(); err != nil {
			log.Printf("Error updating rate limiter configuration in Consul: %v", err)
		}
	}

	if rl.Tokens > 0 {
		rl.Tokens--
		return true // Allow the request
	}

	return false // Rate limit exceeded
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rl *RateLimiter) storeRateLimiterInConsul() error {
	rl.mu.Lock() // Lock before accessing shared state
	defer rl.mu.Unlock()

	// Store the rate limiter state in Consul
	// Modify this function to store the specific fields you want to save
	// In this example, we'll save "Tokens" and "LastRefilled" as JSON
	value, err := json.Marshal(map[string]interface{}{
		"Tokens":       rl.Tokens,
		"LastRefilled": rl.LastRefilled,
	})
	if err != nil {
		return err
	}

	pair := &api.KVPair{
		Key:   rl.Key,
		Value: value,
	}

	_, err = rl.KV.Put(pair, nil)
	return err
}

func main() {
	// Create a Consul client
	config := api.DefaultConfig()
	client, err := api.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	// Key-Value store interaction
	kv := client.KV()
	fmt.Println("----------2")

	// Define the Consul key for the rate limiter configuration
	key := "my-rate-limiter"

	// Create a rate limiter and fetch its initial configuration from Consul
	rateLimiter, err := NewRateLimiter(kv, key)
	fmt.Println("----------3")

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("----------4")

	// Simulate requests
	for i := 1; i <= 50; i++ {
		if rateLimiter.Allow() {
			fmt.Printf("Request %d: Allowed\n", i)
		} else {
			fmt.Printf("Request %d: Rate limit exceeded\n", i)
		}

		time.Sleep(34 * time.Millisecond) // Simulate request spacing
	}
}
