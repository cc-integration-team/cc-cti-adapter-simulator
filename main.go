package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	pkgredis "github.com/cc-integration-team/cc-pkg/pkg/redis"
	"github.com/redis/go-redis/v9"
)

type CTIEvent struct {
	Topic string       `json:"topic"`
	Body  CTIEventBody `json:"body"`
}

type CTIEventBody struct {
	EventName     string            `json:"eventName"`
	CallID        string            `json:"callID"`
	Extension     string            `json:"extension"`
	AgentID       string            `json:"agentID"`
	AgentLogin    string            `json:"agentLogin"`
	CustomerPhone string            `json:"customerPhone"`
	Direction     string            `json:"direction"`
	CallerID      string            `json:"callerID"`
	CalleeID      string            `json:"calleeID"`
	Metadata      map[string]string `json:"metadata"`
}

var (
	isRunning bool
	ticker    *time.Ticker
	stopChan  chan struct{}
)

func main() {
	config, err := LoadConfig("./config")
	if err != nil {
		log.Fatal(err)
	}

	redisClient, err := pkgredis.NewRedisClient(config.Redis)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	stopChan = make(chan struct{})

	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		if isRunning {
			log.Println("🛑 Trigger already running, stopping now...")
			stopChan <- struct{}{}
			isRunning = false
			w.Write([]byte("Stopped pushing events"))
			return
		}

		isRunning = true
		log.Println("▶️ Starting to push events every", config.Push.Interval)
		ticker = time.NewTicker(config.Push.Interval)

		go func() {
			for {
				select {
				case <-ticker.C:
					log.Println("🚀 Pushing phone range 0000001000 -> 0000009999 ...")
					pushEvents(redisClient)
				case <-stopChan:
					log.Println("🛑 Received stop signal")
					ticker.Stop()
					return
				}
			}
		}()

		w.Write([]byte("Started looping phone 0000001000 -> 0000009999"))
	})

	log.Println("✅ Listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func pushEvents(redisClient *redis.Client) {
	var wg sync.WaitGroup

	for i := 1000; i <= 1999; i++ {
		phone := fmt.Sprintf("%010d", i) // 10 digits: 0000001000 -> 0000001999

		event := CTIEvent{
			Topic: "agent",
			Body: CTIEventBody{
				EventName:     "connected",
				CallID:        fmt.Sprintf("call-%d", i),
				Extension:     fmt.Sprintf("10%02d", i%10),
				AgentID:       fmt.Sprintf("agent-%d", i),
				AgentLogin:    fmt.Sprintf("login-%d", i),
				CustomerPhone: phone,
				Direction:     "Callin",
				CallerID:      fmt.Sprintf("caller-%d", i),
				CalleeID:      fmt.Sprintf("callee-%d", i),
				Metadata: map[string]string{
					"system": "auto-sender",
					"time":   time.Now().Format(time.RFC3339),
				},
			},
		}

		wg.Add(1)
		go func(ev CTIEvent) {
			defer wg.Done()
			payload, err := json.Marshal(ev)
			if err != nil {
				log.Printf("❌ Marshal error: %v", err)
				return
			}

			err = redisClient.Publish(context.Background(), "cti_event:cisco:jtapi", payload).Err()
			if err != nil {
				log.Printf("❌ Redis publish error: %v", err)
			}
		}(event)
	}

	wg.Wait()
	log.Println("✅ Done pushing phones 0000001000 -> 0000009999")
}
