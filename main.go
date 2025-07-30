package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	pkgredis "github.com/cc-integration-team/cc-pkg/pkg/redis"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Event struct {
	EventName string  `json:"eventName"`
	EventID   string  `json:"eventID"`
	CTITag    string  `json:"ctiTag"`
	Details   Details `json:"details"`
}

type Details struct {
	CTICallID string `json:"cticallid"`
	Ext       string `json:"ext"`
	Customer  string `json:"customer"`
	AgentID   string `json:"agentID"`
	CallerID  string `json:"callerid"`
	CalleeID  string `json:"calleeid"`
	Direction string `json:"direction"`
	QueueName string `json:"queueName"`
	Language  string `json:"language"`
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
		ticker = time.NewTicker(config.Push.Interval)

		go func() {
			for {
				select {
				case <-ticker.C:
					log.Printf("🚀 Pushing %d connected events...\n", config.Push.Count)
					pushEvents(redisClient, config.Push.Count)
				case <-stopChan:
					log.Println("🛑 Received stop signal")
					ticker.Stop()
					return
				}
			}
		}()

		w.Write([]byte(fmt.Sprintf("Started pushing %d events every %s", config.Push.Count, config.Push.Interval)))
	})

	log.Println("✅ Listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func pushEvents(redisClient *redis.Client, count int) {
	var wg sync.WaitGroup

	serialLen := len(strconv.Itoa(count))
	prefixLen := 10 - serialLen
	prefix := ""
	for i := 0; i < prefixLen; i++ {
		prefix += "0"
	}

	for i := 0; i < count; i++ {
		serial := fmt.Sprintf("%0*d", serialLen, i)
		phone := prefix + serial

		event := Event{
			EventName: "connected",
			EventID:   uuid.New().String(),
			CTITag:    "CISCO_JTAPI",
			Details: Details{
				CTICallID: fmt.Sprintf("cticallid-%d", i),
				Ext:       fmt.Sprintf("10%02d", i%10),
				AgentID:   fmt.Sprintf("agent-%d", i),
				Customer:  phone,
				Direction: "Callin",
				CallerID:  fmt.Sprintf("caller-%d", i),
				CalleeID:  fmt.Sprintf("callee-%d", i),
				QueueName: fmt.Sprintf("queue-%d", i%5),
				Language:  "vi",
			},
		}

		wg.Add(1)
		go func(ev Event) {
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
	log.Printf("✅ Done pushing %d connected events", count)
}
