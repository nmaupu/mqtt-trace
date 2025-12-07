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

// FileWriter handles writing messages to the output file
type FileWriter struct {
	mu       sync.Mutex
	filePath string
}

// NewFileWriter creates a new file writer
func NewFileWriter(filePath string) *FileWriter {
	return &FileWriter{
		filePath: filePath,
	}
}

// WriteMessage appends a message to the output file in the format: <date>|name=<name>|rssi=<rssi>
func (fw *FileWriter) WriteMessage(payload map[string]any) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(fw.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// Build the output line: <date>|name=<name>|rssi=<rssi>
	date := time.Now().Format(time.RFC3339)
	line := date

	// Add name if present
	if name, ok := payload["name"]; ok {
		line += fmt.Sprintf("|name=%v", name)
	}

	// Add rssi if present
	if rssi, ok := payload["rssi"]; ok {
		line += fmt.Sprintf("|rssi=%v", rssi)
	}

	// Write line with newline
	line += "\n"
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// messageHandler handles incoming MQTT messages
func messageHandler(writer *FileWriter) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		var payload map[string]any
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			log.Printf("Error unmarshaling message from topic %s: %v", msg.Topic(), err)
			return
		}

		if err := writer.WriteMessage(payload); err != nil {
			log.Printf("Error saving message: %v", err)
			return
		}

		log.Printf("Received message on topic %s", msg.Topic())
	}
}

// loadConfig loads configuration from file using Viper
func loadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("mqtt.port", 1883)
	viper.SetDefault("output_file", "mqtt-trace.log")

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

	// Create file writer
	writer := NewFileWriter(config.OutputFile)

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
	opts.SetDefaultPublishHandler(messageHandler(writer))

	// Create and start MQTT client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Println("Connected to MQTT broker")

	// Subscribe to all topics
	for _, topic := range config.MQTT.Topics {
		if token := client.Subscribe(topic, 0, messageHandler(writer)); token.Wait() && token.Error() != nil {
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
}
