// notifiers/email/email.go
package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"server-monitor/collectors"
)

// EmailNotifier implements the Notifier interface for email notifications
type EmailNotifier struct {
	from       string
	to         []string
	smtpServer string
	smtpPort   int
	username   string
	password   string
	auth       smtp.Auth
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{}
}

// Name returns the name of the notifier
func (n *EmailNotifier) Name() string {
	return "email"
}

// Init initializes the email notifier with configuration
func (n *EmailNotifier) Init(config map[string]interface{}) error {
	var ok bool

	// Get from address
	if n.from, ok = config["from"].(string); !ok {
		return fmt.Errorf("missing 'from' in email config")
	}

	// Get to addresses
	_, exists := config["to"]
	if !exists {
		return fmt.Errorf("missing 'to' in email config")
	}

	toRaw, ok := config["to"].([]string)
	if !ok {
		return fmt.Errorf("'to' field must be an array of email addresses")
	}

	n.to = append(n.to, toRaw...)

	if len(n.to) == 0 {
		return fmt.Errorf("no valid 'to' addresses in email config")
	}

	// Get SMTP server
	if n.smtpServer, ok = config["smtp_server"].(string); !ok {
		return fmt.Errorf("missing 'smtp_server' in email config")
	}

	// Get SMTP port
	portRaw, ok := config["smtp_port"].(int)
	if !ok {
		return fmt.Errorf("missing 'smtp_port' in email config")
	}
	n.smtpPort = portRaw

	// Get username and password (optional if SMTP server doesn't require auth)
	n.username, _ = config["username"].(string)
	n.password, _ = config["password"].(string)

	// Set up authentication if credentials are provided
	if n.username != "" && n.password != "" {
		host := n.smtpServer
		n.auth = smtp.PlainAuth("", n.username, n.password, host)
	}

	return nil
}

// Notify sends an email notification for the provided results
func (n *EmailNotifier) Notify(ctx context.Context, results []collectors.Result) error {
	// Filter only unhealthy results
	var unhealthyResults []collectors.Result
	for _, result := range results {
		if !result.IsHealthy {
			unhealthyResults = append(unhealthyResults, result)
		}
	}

	// Skip if no unhealthy results
	if len(unhealthyResults) == 0 {
		return nil
	}

	// Prepare email content
	subject := fmt.Sprintf("Server Alert: %d issue(s) detected", len(unhealthyResults))
	body := n.formatEmailBody(unhealthyResults)

	// Compose the email
	header := make(map[string]string)
	header["From"] = n.from
	header["To"] = strings.Join(n.to, ", ")
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the server, authenticate, and send the email
	addr := fmt.Sprintf("%s:%d", n.smtpServer, n.smtpPort)

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing
	}

	// Send the email
	var err error
	if n.auth != nil {
		err = smtp.SendMail(addr, n.auth, n.from, n.to, []byte(message))
	} else {
		// Connect to the server
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer client.Close()

		// Set the sender and recipients
		if err := client.Mail(n.from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, addr := range n.to {
			if err := client.Rcpt(addr); err != nil {
				return fmt.Errorf("failed to set recipient: %w", err)
			}
		}

		// Send the email body
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to start email data: %w", err)
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write email body: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close email data: %w", err)
		}

		_ = client.Quit()
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// formatEmailBody creates a formatted message body for the email
func (n *EmailNotifier) formatEmailBody(results []collectors.Result) string {
	var builder strings.Builder

	builder.WriteString("The following issues were detected on the server:\n\n")

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("%d. [%s] %s\n",
			i+1,
			result.Timestamp.Format(time.RFC1123),
			result.Message))

		// Add metrics if available
		if len(result.Metrics) > 0 {
			builder.WriteString("   Metrics:\n")
			for key, value := range result.Metrics {
				builder.WriteString(fmt.Sprintf("   - %s: %.2f\n", key, value))
			}
		}

		builder.WriteString("\n")
	}

	builder.WriteString("\n--\n")
	builder.WriteString("This is an automated message from the server monitoring system.\n")
	builder.WriteString("Please do not reply to this email.\n")

	return builder.String()
}

// Close performs any necessary cleanup
func (n *EmailNotifier) Close() error {
	// No cleanup needed for email notifier
	return nil
}
