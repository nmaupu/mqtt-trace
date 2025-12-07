# MQTT Trace

A lightweight Go application designed to monitor and record MQTT message reception intervals, specifically tailored for Xiaomi LYSD03MMC temperature/humidity sensors (though it works with any MQTT device that publishes JSON payloads).

## Overview

This tool subscribes to MQTT topics and records the timestamp of each received message, along with filtered payload data. It's particularly useful for analyzing the reporting intervals of IoT sensors like the Xiaomi LYSD03MMC, helping you understand how frequently your devices are sending data.

## Features

- **MQTT Subscription**: Subscribe to multiple MQTT topics simultaneously
- **Timestamp Tracking**: Records the exact time (RFC3339 format) when each message is received
- **Payload Filtering**: Extracts only relevant fields (`rssi` and `name`) from incoming messages
- **JSON Output**: Saves all records in a structured JSON format
- **Real-time Updates**: Output file is updated immediately upon receiving each message
- **Graceful Shutdown**: Handles interrupt signals (Ctrl+C) cleanly

## Requirements

- Go 1.25.1 or later
- Access to an MQTT broker
- MQTT credentials (username/password)

## Installation

1. Clone or download this repository

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build -o mqtt-trace
   ```

## Configuration

1. Copy the example configuration file:
   ```bash
   cp config.yaml.example config.yaml
   ```

2. Edit `config.yaml` with your MQTT broker settings:

   ```yaml
   mqtt:
     broker: localhost          # MQTT broker address
     port: 1883                  # MQTT broker port
     username: your_username     # MQTT username
     password: your_password     # MQTT password
     topics:
       - "+/+/BTtoMQTT/A4C138DBBC6F"  # Topic pattern for sensor 1
       - "+/+/BTtoMQTT/A4C138C3A050"  # Topic pattern for sensor 2
   
   output_file: "mqtt-trace.json"    # Output JSON file path
   ```

### Topic Patterns for Xiaomi LYSD03MMC

The Xiaomi LYSD03MMC sensors typically publish to MQTT topics following the pattern:
```
+/+/BTtoMQTT/<MAC_ADDRESS>
```

Where:
- `+` is a wildcard matching any single level
- `<MAC_ADDRESS>` is the MAC address of your sensor (e.g., `A4C138DBBC6F`)

You can add multiple sensors by listing their topic patterns in the `topics` array.

## Usage

Run the application:

```bash
go run main.go
```

Or if you've built the binary:

```bash
./mqtt-trace
```

To specify a custom configuration file:

```bash
go run main.go /path/to/config.yaml
```

The application will:
1. Connect to the MQTT broker
2. Subscribe to all configured topics
3. Start logging received messages
4. Save messages to the output JSON file in real-time

Press `Ctrl+C` to stop the application gracefully. The program will disconnect from the MQTT broker and display the total number of messages recorded.

## Output Format

The output JSON file contains an array of message records. Each record includes:

- **`date`**: ISO 8601 timestamp (RFC3339) when the message was received
- **`payload`**: Filtered payload containing only `rssi` and `name` fields (if present in the original message)

Example output (`mqtt-trace.json`):

```json
[
  {
    "date": "2024-01-15T10:30:45Z",
    "payload": {
      "rssi": -65,
      "name": "LYSD03MMC"
    }
  },
  {
    "date": "2024-01-15T10:31:15Z",
    "payload": {
      "rssi": -67,
      "name": "LYSD03MMC"
    }
  }
]
```

## Analyzing Intervals

To analyze the intervals between messages, you can:

1. Use the timestamps in the `date` field to calculate time differences
2. Import the JSON file into a data analysis tool (Python pandas, Excel, etc.)
3. Write a simple script to parse the JSON and compute intervals

Example Python snippet to calculate intervals:

```python
import json
from datetime import datetime

with open('mqtt-trace.json') as f:
    records = json.load(f)

for i in range(1, len(records)):
    prev_time = datetime.fromisoformat(records[i-1]['date'].replace('Z', '+00:00'))
    curr_time = datetime.fromisoformat(records[i]['date'].replace('Z', '+00:00'))
    interval = (curr_time - prev_time).total_seconds()
    print(f"Interval: {interval} seconds")
```

## About Xiaomi LYSD03MMC

The Xiaomi LYSD03MMC is a Bluetooth Low Energy (BLE) temperature and humidity sensor. When integrated with an MQTT bridge (like BTtoMQTT or similar), it publishes sensor data to MQTT topics. This tool helps you monitor:

- How frequently the sensor reports data
- Signal strength (RSSI) variations over time
- Message reception reliability

## Compatibility

While designed for Xiaomi LYSD03MMC sensors, this tool works with any MQTT device that publishes JSON payloads. The payload filtering will extract `rssi` and `name` fields if they exist in the message, otherwise the payload will be empty.

## License

This project is provided as-is for personal use.
