package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventResponse struct {
	Status    string      `json:"status"`
	Partition int         `json:"partition"`
	Offset    int64       `json:"offset"`
	Event     interface{} `json:"event"`
}

type MovieEvent struct {
	MovieID     int      `json:"movie_id"`
	Title       string   `json:"title"`
	Action      string   `json:"action"`
	UserID      *int     `json:"user_id,omitempty"`
	Rating      *float64 `json:"rating,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Description *string  `json:"description,omitempty"`
}

type UserEvent struct {
	UserID    int       `json:"user_id"`
	Username  *string   `json:"username,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

type PaymentEvent struct {
	PaymentID int       `json:"payment_id"`
	UserID    int       `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Method    *string   `json:"method_type,omitempty"`
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func brokers() []string {
	return []string{env("KAFKA_BROKERS", "kafka:9092")}
}

func main() {
	mux := http.NewServeMux()

	// health
	mux.HandleFunc("/api/events/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"status": true})
	})

	// endpoints
	mux.HandleFunc("/api/events/movie", handleMovie)
	mux.HandleFunc("/api/events/user", handleUser)
	mux.HandleFunc("/api/events/payment", handlePayment)

	go backgroundConsume("movie-events")
	go backgroundConsume("user-events")
	go backgroundConsume("payment-events")

	port := env("PORT", "8082")
	log.Printf("[events] listening :%s | brokers=%v", port, brokers())
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func handleMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var e MovieEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := produceAndAwait("movie-events", e)
	writeJSON(w, resp, err)
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var e UserEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := produceAndAwait("user-events", e)
	writeJSON(w, resp, err)
}

func handlePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var e PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := produceAndAwait("payment-events", e)
	writeJSON(w, resp, err)
}

func produceAndAwait(topic string, payload any) (*EventResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	val, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers(),
		Topic:       topic,
		StartOffset: kafka.LastOffset, // -1
		MaxWait:     500 * time.Millisecond,
	})
	defer reader.Close()

	// writer (sync)
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers()...),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		Balancer:               &kafka.LeastBytes{},
	}
	defer writer.Close()

	if err := writer.WriteMessages(ctx, kafka.Message{
		Time:  time.Now().UTC(),
		Value: val,
	}); err != nil {
		return nil, fmt.Errorf("kafka write: %w", err)
	}

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("kafka read back: %w", err)
	}

	log.Printf("[events][%s] consumed back: partition=%d offset=%d value=%s",
		topic, msg.Partition, msg.Offset, string(msg.Value))

	return &EventResponse{
		Status:    "success",
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Event:     payload,
	}, nil
}

func backgroundConsume(topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers(),
		Topic:   topic,
		GroupID: "events-service",
		MaxWait: 500 * time.Millisecond,
	})
	defer r.Close()

	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[events][bg][%s] read error: %v", topic, err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[events][bg][%s] partition=%d offset=%d value=%s", topic, msg.Partition, msg.Offset, string(msg.Value))
	}
}

func writeJSON(w http.ResponseWriter, resp *EventResponse, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
