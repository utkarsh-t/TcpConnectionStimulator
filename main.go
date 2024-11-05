package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// Config structure for TPS configuration
type Config struct {
	TPS []struct {
		Port int `yaml:"port"`
		Rate int `yaml:"rate"`
	} `yaml:"tps"`
}

// RedisStore represents an in-memory storage like Redis
// I Just created a mock redis so that the code can be run anywhere without specififcally requiring redis server

type RedisStore struct {
	data map[string]string
	mu   sync.RWMutex
}

//key(int):key(int)

//key:randInt

// NewRedisStore initializes a new Redis-like store with preloaded records

func NewRedisStore(numRecords int) *RedisStore {
	store := &RedisStore{data: make(map[string]string)}
	for i := 0; i < numRecords; i++ {
		key := fmt.Sprintf("record-%d", i)
		// 34:34 77:77
		store.Set(key, key) // Store the key-value pair where value is same as key
	}
	return store
}

// Set stores a key-value pair in RedisStore
func (r *RedisStore) Set(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
}

// Get retrieves a value from RedisStore by key
func (r *RedisStore) Get(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.data[key]
	return val, ok
}

// GetUnprocessedRecord retrieves the next record without an ID (unprocessed).
func (r *RedisStore) GetUnprocessedRecord() (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, value := range r.data {
		if value == key { // Check if the value matches the key, meaning it's unprocessed
			return key, true
		}
	}
	return "", false
}

// LoadConfig loads the TPS config from a YAML file
func loadConfig(filename string) (*Config, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// TCP Server setup: listens on specified port and handles requests
func startServer(port int, wg *sync.WaitGroup) {
	defer wg.Done()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Error on port %d: %v", port, err)
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go func(c net.Conn) {
			defer c.Close()
			handleConnection(c)
		}(conn)
	}
}

// handleConnection generates a unique response for each client request
func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		_, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		resp := fmt.Sprintf("ID-%d\n", rand.Int())
		conn.Write([]byte(resp))
	}
}

// ClientTask represents a task for a client connection
type ClientTask struct {
	Port  int
	Rate  int
	Store *RedisStore
}

// Worker function to process client tasks
func worker(id int, tasks <-chan ClientTask, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	for task := range tasks {
		sem <- struct{}{} // Acquire a token before starting the client
		go func(t ClientTask) {
			defer func() { <-sem }() // Release the token when done
			startClient(t.Port, t.Rate, t.Store)
			fmt.Printf("Worker %d completed task for port %d\n", id, t.Port)
		}(task)
	}
}

func startClient(port, rate int, store *RedisStore) {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("Client error on port %d: %v", port, err)
		return
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	ticker := time.NewTicker(time.Second / time.Duration(rate))
	defer ticker.Stop()

	// Continue until all records have received an ID
	for {
		<-ticker.C

		// Retrieve the next unprocessed record
		key, found := store.GetUnprocessedRecord()
		if !found {
			fmt.Println("All records processed")
			return
		}

		// Send the record key to the server
		writer.WriteString(key + "\n")
		writer.Flush()

		// Read server response
		resp, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		// Store the response ID in RedisStore
		store.Set(key, strings.TrimSpace(resp)) // Mark as processed
		fmt.Printf("Stored: %s -> %s\n", key, resp)
	}
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	store := NewRedisStore(5000000) // Preload with 5 million records
	var wg sync.WaitGroup

	// Semaphore channel to limit the number of concurrent clients1

	sem := make(chan struct{}, 10) // Buffered channel with a capacity of 10

	// Channel for client tasks
	tasks := make(chan ClientTask)

	// Create a fixed number of workers
	for w := 1; w <= 5; w++ {
		wg.Add(1)
		go worker(w, tasks, &wg, sem)
	}

	for _, t := range config.TPS {
		wg.Add(1)
		go startServer(t.Port, &wg)
	}

	time.Sleep(1 * time.Second) // Allow servers to start

	// Submit tasks to the worker pool
	for _, t := range config.TPS {
		tasks <- ClientTask{Port: t.Port, Rate: t.Rate, Store: store}
	}

	close(tasks) // Close the tasks channel to indicate no more tasks

	wg.Wait() // Wait for all workers to finish
}
