High-Throughput TCP Client-Server Simulation in Go
This project demonstrates a high-throughput TCP client-server application in Go. It simulates an in-memory Redis-like store containing 5 million records, processes these records through a multi-client, multi-server architecture, and implements concurrency control using Goroutines, worker pools, and channels.

Features
Redis-like In-Memory Store: Stores 5 million records, marking records as processed when they receive a unique server ID.
Rate-Limited Client: Configurable TPS rate per client that enforces controlled throughput.
Multi-Connection TCP Server: Listens on multiple ports and returns a unique ID for each request.
Concurrency Management: Uses worker pools, semaphores, and Goroutines to limit connections and enforce rate limits.

Configuration
The configuration file (config.yaml) defines the Transactions Per Second (TPS) rate and ports for each client-server pair.

Example config.yaml:

yaml

tps:
  - port: 8001
    rate: 100
  - port: 8002
    rate: 200
  # Add more configurations as needed

  
In this configuration:

port specifies the port each server listens on.
rate defines the TPS (Transactions Per Second) rate each client should adhere to for that server.
Project Structure
main.go: Contains the main logic for initializing the server, clients, and Redis-like in-memory storage.
config.yaml: The configuration file defining TPS for each client.

Code Overview
1. Redis-like In-Memory Storage (RedisStore)
RedisStore simulates a Redis database to hold and manage 5 million records. It offers:

Initialization: Preloads 5 million records.
GetUnprocessedRecord: Retrieves records that haven’t been processed (i.e., those without a server ID).
Set: Updates records with a unique server ID upon processing.
2. Server (startServer)
Each server listens on a specified port from config.yaml, handling incoming TCP connections.

Concurrency: Each connection is handled by a Goroutine.
Response Generation: For each request, the server generates a unique ID and sends it back to the client.
3. Client (startClient)
The client reads TPS settings from config.yaml, connects to the corresponding server, and processes records at the configured TPS rate.

Rate Limiting: Uses a ticker to control the frequency of record processing based on TPS.
Concurrency Control: A semaphore (sem) limits the maximum concurrent connections to 10.
Mapping Responses: Each response from the server is stored back in RedisStore, marking the record as processed.
4. Main Function (main)
In main, we set up all servers and clients based on config.yaml.

Server Initialization: Starts a server on each specified port.
Client Initialization: Launches a client for each configuration entry, using a semaphore to limit connections.
Code Walkthrough
RedisStore Initialization

store := NewRedisStore()

This function loads 5 million records with keys like "record-1", "record-2", etc.

Starting Servers
Each server runs on a separate Goroutine and listens on the configured port:

for _, t := range config.TPS {
	wg.Add(1)
	go startServer(t.Port, &wg)
}


Client Setup with Semaphore
Each client sends a maximum of 100 requests and uses the semaphore to ensure no more than 10 concurrent connections.


sem := make(chan struct{}, 10)
for _, t := range config.TPS {
	wg.Add(1)
	go func(port, rate int) {
		
  sem <- struct{}{}
		defer func() { <-sem }()
		startClient(port, rate, store, sem, &wg)
	
 }(t.Port, t.Rate)
}

Additional Information
Concurrency & Rate Limiting
This project achieves concurrency and rate-limiting by:

Using Goroutines for parallel handling of clients and connections.
Worker Pool Pattern using a semaphore channel to limit concurrent client connections to 10.
Rate Limiting using a ticker to control TPS rate for each client.
Error Handling
Basic error handling is implemented in the server and client, logging any connection or read/write errors for easier debugging.

**Future Improvements**

**Persistent Storage:** Implement a database or actual Redis backend for large-scale data persistence.

**Enhanced Rate Limiting**: Use token-bucket algorithms for more precise TPS control.

**Configuration Enhancement**s: Allow dynamic configuration reloads without restarting the application.

Conclusion

This project demonstrates a robust high-throughput, multi-client TCP application in Go, with effective concurrency management and rate-limiting. Thank you for checking out this project!
