# Lightweight Server Monitoring System

A simple, lightweight monitoring system that can track server resources like disk space and memory usage, with configurable thresholds and email notifications.

## Features

- **Modular Design**: Clear separation between collectors, monitoring service, and notification
- **Asynchronous Operation**: Collectors run independently at different intervals
- **Extensible**: Easy to add new collectors or notification methods
- **Configuration-driven**: All thresholds and behaviors are configurable
- **Lightweight**: No database or web server dependencies

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Access to an SMTP server for email notifications (optional)

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/server-monitor.git
   cd server-monitor
   ```

2. Build the application:

   ```bash
   go build -o server-monitor
   ```

3. Create your configuration file:

   ```bash
   cp config.yaml.example config.yaml
   ```

4. Edit the configuration file to set your thresholds and notification settings.

5. Run the monitor:

   ```bash
   ./server-monitor -config config.yaml
   ```

## Configuration

The monitoring system is driven by a YAML configuration file. Here's an example:

```yaml
monitor:
  default_interval_seconds: 300  # Default check interval: 5 minutes

collectors:
  disk_space:
    enabled: true
    interval_seconds: 60  # Check disk space every minute
    settings:
      paths:
        - path: "/"
          threshold_gb: 5
          threshold_percent: 90
        - path: "/home"
          threshold_gb: 10
          threshold_percent: 85

  memory:
    enabled: true
    interval_seconds: 120  # Check memory every 2 minutes
    settings:
      threshold_percent: 90

notifications:
  email:
    enabled: true
    from: "monitor@example.com"
    to: 
      - "admin@example.com"
    smtp_server: "smtp.example.com"
    smtp_port: 587
    username: "monitor@example.com"
    password: "your-password-here"
```

### Collector Settings

#### Disk Space Collector

- `paths`: List of paths to monitor
  - `path`: Directory path to monitor
  - `threshold_gb`: Alert when free space falls below this amount (in GB)
  - `threshold_percent`: Alert when used space exceeds this percentage

#### Memory Collector

- `threshold_percent`: Alert when memory usage exceeds this percentage

### Notification Settings

#### Email Notifications

- `from`: Sender email address
- `to`: List of recipient email addresses
- `smtp_server`: SMTP server hostname
- `smtp_port`: SMTP server port
- `username`: SMTP authentication username
- `password`: SMTP authentication password

## Adding New Collectors

To add a new collector:

1. Create a new package in the `collectors` directory
2. Implement the `Collector` interface
3. Register the collector in `monitor.registerCollectors()`
4. Add configuration options to the config file

## Adding New Notification Methods

To add a new notification method:

1. Create a new package in the `notifiers` directory
2. Implement the `Notifier` interface
3. Register the notifier in `monitor.registerNotifiers()`
4. Add configuration options to the config file

## License

This project is licensed under the MIT License - see the LICENSE file for details.
