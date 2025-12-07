# MQTT Trace

A lightweight Go application designed to monitor and record MQTT message reception intervals, specifically tailored for Xiaomi LYSD03MMC temperature/humidity sensors (though it works with any MQTT device that publishes JSON payloads).

## Overview

This tool subscribes to MQTT topics and records the timestamp of each received message, along with filtered payload data. It's particularly useful for analyzing the reporting intervals of IoT sensors like the Xiaomi LYSD03MMC, helping you understand how frequently your devices are sending data.

## Features

- **MQTT Subscription**: Subscribe to multiple MQTT topics simultaneously
- **Timestamp Tracking**: Records the exact time (RFC3339 format) when each message is received
- **Payload Filtering**: Extracts only relevant fields (`rssi` and `name`) from incoming messages
- **Line-based Output**: Saves records in a simple, append-only line format for efficient processing
- **Real-time Updates**: Output file is updated immediately upon receiving each message
- **Memory Efficient**: No in-memory storage - messages are written directly to disk
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
   
   output_file: "mqtt-trace.log"    # Output log file path
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
4. Save messages to the output log file in real-time (one line per message)

Press `Ctrl+C` to stop the application gracefully. The program will disconnect from the MQTT broker.

## Output Format

The output log file uses a simple line-based format. Each message is written as a single line appended to the file:

```
<date>|name=<name>|rssi=<rssi>
```

Where:
- **`date`**: ISO 8601 timestamp (RFC3339) when the message was received
- **`name`**: Device name (only included if present in the message)
- **`rssi`**: Signal strength (only included if present in the message)

Example output (`mqtt-trace.log`):

```
2024-01-15T10:30:45Z|name=LYSD03MMC|rssi=-65
2024-01-15T10:31:15Z|name=LYSD03MMC|rssi=-67
2024-01-15T10:31:45Z|name=LYSD03MMC|rssi=-66
```

**Note**: If `name` or `rssi` fields are not present in the message, they will be omitted from the output line. The file is appended to, so it grows over time without truncation.

## Analyzing Intervals

To analyze the intervals between messages, you can parse the log file line by line. Here's an example Python script:

```python
import re
from datetime import datetime

def parse_log_line(line):
    """Parse a log line: <date>|name=<name>|rssi=<rssi>"""
    parts = line.strip().split('|')
    if not parts:
        return None
    
    date_str = parts[0]
    name = None
    rssi = None
    
    for part in parts[1:]:
        if part.startswith('name='):
            name = part[5:]
        elif part.startswith('rssi='):
            rssi = part[5:]
    
    try:
        date = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        return {'date': date, 'name': name, 'rssi': rssi}
    except:
        return None

# Read and parse the log file
records = []
with open('mqtt-trace.log') as f:
    for line in f:
        record = parse_log_line(line)
        if record:
            records.append(record)

# Calculate intervals
for i in range(1, len(records)):
    prev_time = records[i-1]['date']
    curr_time = records[i]['date']
    interval = (curr_time - prev_time).total_seconds()
    print(f"Interval: {interval} seconds (RSSI: {records[i]['rssi']})")
```

You can also use command-line tools like `awk` or `grep` to process the log file, or import it into data analysis tools that support line-delimited formats.

## About Xiaomi LYSD03MMC

The Xiaomi LYSD03MMC is a Bluetooth Low Energy (BLE) temperature and humidity sensor. When integrated with an MQTT bridge (like BTtoMQTT or similar), it publishes sensor data to MQTT topics. This tool helps you monitor:

- How frequently the sensor reports data
- Signal strength (RSSI) variations over time
- Message reception reliability

## Compatibility

While designed for Xiaomi LYSD03MMC sensors, this tool works with any MQTT device that publishes JSON payloads. The payload filtering will extract `rssi` and `name` fields if they exist in the message. If a field is not present, it will simply be omitted from the output line.

## License

This project is provided as-is for personal use.
