package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	MQTT struct {
		Broker   string   `mapstructure:"broker"`
		Port     int      `mapstructure:"port"`
		Username string   `mapstructure:"username"`
		Password string   `mapstructure:"password"`
		Topics   []string `mapstructure:"topics"`
	} `mapstructure:"mqtt"`
	OutputFile string `mapstructure:"output_file"`
}

// MessageRecord represents a single message record in the output file
type MessageRecord struct {
	Date    string                 `json:"date"`
	Payload map[string]interface{} `json:"payload"`
}

// MessageStore manages the collection of messages
type MessageStore struct {
	mu       sync.RWMutex
	messages []MessageRecord
	filePath string
}

// NewMessageStore creates a new message store
func NewMessageStore(filePath string) *MessageStore {
	return &MessageStore{
		messages: make([]MessageRecord, 0),
		filePath: filePath,
	}
}

// AddMessage adds a new message to the store and saves to file
func (ms *MessageStore) AddMessage(payload map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Filter payload to only include rssi and name fields
	filteredPayload := make(map[string]interface{})
	if rssi, ok := payload["rssi"]; ok {
		filteredPayload["rssi"] = rssi
	}
	if name, ok := payload["name"]; ok {
		filteredPayload["name"] = name
	}

	record := MessageRecord{
		Date:    time.Now().Format(time.RFC3339),
		Payload: filteredPayload,
	}

	ms.messages = append(ms.messages, record)

	// Save to file
	return ms.saveToFile()
}

// saveToFile writes all messages to the JSON file
func (ms *MessageStore) saveToFile() error {
	file, err := os.Create(ms.filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ms.messages); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// messageHandler handles incoming MQTT messages
func messageHandler(store *MessageStore) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			log.Printf("Error unmarshaling message from topic %s: %v", msg.Topic(), err)
			return
		}

		if err := store.AddMessage(payload); err != nil {
			log.Printf("Error saving message: %v", err)
			return
		}

		log.Printf("Received message on topic %s: %+v", msg.Topic(), payload)
	}
}

// loadConfig loads configuration from file using Viper
func loadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("mqtt.port", 1883)
	viper.SetDefault("output_file", "mqtt-trace.json")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if config.MQTT.Broker == "" {
		return nil, fmt.Errorf("mqtt.broker is required")
	}
	if len(config.MQTT.Topics) == 0 {
		return nil, fmt.Errorf("at least one mqtt.topic is required")
	}

	return &config, nil
}

func main() {
	// Load configuration
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Loaded configuration from %s", configPath)
	log.Printf("MQTT Broker: %s:%d", config.MQTT.Broker, config.MQTT.Port)
	log.Printf("Output file: %s", config.OutputFile)
	log.Printf("Subscribing to %d topics", len(config.MQTT.Topics))

	// Create message store
	store := NewMessageStore(config.OutputFile)

	// Setup MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.MQTT.Broker, config.MQTT.Port))
	opts.SetClientID(fmt.Sprintf("mqtt-trace-%d", time.Now().Unix()))
	opts.SetUsername(config.MQTT.Username)
	opts.SetPassword(config.MQTT.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	// Set default message handler
	opts.SetDefaultPublishHandler(messageHandler(store))

	// Create and start MQTT client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Println("Connected to MQTT broker")

	// Subscribe to all topics
	for _, topic := range config.MQTT.Topics {
		if token := client.Subscribe(topic, 0, messageHandler(store)); token.Wait() && token.Error() != nil {
			log.Fatalf("Failed to subscribe to topic %s: %v", topic, token.Error())
		}
		log.Printf("Subscribed to topic: %s", topic)
	}

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("MQTT trace started. Press Ctrl+C to stop...")
	<-sigChan

	log.Println("Shutting down...")
	client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
	log.Printf("Total messages recorded: %d", len(store.messages))
}
