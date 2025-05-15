// monitor/monitor.go
package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"server-monitor/collectors"
	"server-monitor/collectors/disk"
	"server-monitor/collectors/memory"
	"server-monitor/config"
	"server-monitor/notifiers"
	"server-monitor/notifiers/email"

	"go.uber.org/zap"
)

// MonitorService is the main service that orchestrates collectors and notifiers
type MonitorService struct {
	config            *config.Config
	collectorRegistry *collectors.Registry
	notifierRegistry  *notifiers.Registry
	collectorTasks    map[string]context.CancelFunc
	logger            *zap.Logger
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.Mutex
}

// NewMonitorService creates a new monitoring service
func NewMonitorService(logger *zap.Logger, cfg *config.Config) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	return &MonitorService{
		config:            cfg,
		collectorRegistry: collectors.NewRegistry(logger.Named("collectorRegistry")),
		notifierRegistry:  notifiers.NewRegistry(logger.Named("notifierRegistry")),
		collectorTasks:    make(map[string]context.CancelFunc),
		ctx:               ctx,
		cancel:            cancel,
		logger:            logger,
	}
}

// Start initializes and starts the monitoring service
func (s *MonitorService) Start() error {
	s.logger.Info("Initializing monitoring service...")

	// Register collectors
	if err := s.registerCollectors(); err != nil {
		s.logger.Error("Failed to register collectors", zap.Error(err))
		return err
	}

	// Register notifiers
	if err := s.registerNotifiers(); err != nil {
		s.logger.Error("Failed to register notifiers", zap.Error(err))
		return err
	}

	// Initialize enabled collectors
	if err := s.initializeCollectors(); err != nil {
		s.logger.Error("Failed to initialize collectors", zap.Error(err))
		return err
	}

	// Initialize enabled notifiers
	if err := s.initializeNotifiers(); err != nil {
		s.logger.Error("Failed to initialize notifiers", zap.Error(err))
		return err
	}

	// Start collector tasks
	if err := s.startCollectorTasks(); err != nil {
		s.logger.Error("Failed to start collector tasks", zap.Error(err))
		return err
	}

	s.logger.Info("Monitoring service started successfully")
	return nil
}

// Stop gracefully stops the monitoring service
func (s *MonitorService) Stop() {
	s.logger.Info("Stopping monitoring service...")

	// Cancel main context to signal all tasks to stop
	s.cancel()

	// Wait for all tasks to complete
	s.wg.Wait()

	// Clean up collectors
	for _, c := range s.collectorRegistry.GetAll() {
		if err := c.Cleanup(); err != nil {
			s.logger.Error("Error cleaning up collector", zap.String("collector", c.Name()), zap.Error(err))
		}
	}

	// Clean up notifiers
	for _, n := range s.notifierRegistry.GetAll() {
		if err := n.Close(); err != nil {
			s.logger.Error("Error closing notifier", zap.String("notifier", n.Name()), zap.Error(err))
		}
	}

	s.logger.Info("Monitoring service stopped")
}

// registerCollectors registers all available collectors
func (s *MonitorService) registerCollectors() error {
	// Register disk space collector
	if err := s.collectorRegistry.Register(disk.NewDiskCollector(s.logger.Named("diskCollector"))); err != nil {
		s.logger.Error("Failed to register disk collector", zap.Error(err))
		return err
	}

	// Register memory collector
	if err := s.collectorRegistry.Register(memory.NewMemoryCollector(s.logger.Named("memoryCollector"))); err != nil {
		s.logger.Error("Failed to register memory collector", zap.Error(err))
		return err
	}

	s.logger.Info("Registered collectors", zap.Strings("collectors", s.collectorRegistry.CollectorNames()))
	return nil
}

// registerNotifiers registers all available notifiers
func (s *MonitorService) registerNotifiers() error {
	// Register email notifier
	if err := s.notifierRegistry.Register(email.NewEmailNotifier(s.logger.Named("emailNotifier"))); err != nil {
		s.logger.Error("Failed to register email notifier", zap.Error(err))
		return err
	}

	// Register other notifiers here...

	log.Printf("Registered notifiers: %v", s.notifierRegistry.NotifierNames())
	return nil
}

// initializeCollectors initializes all enabled collectors
func (s *MonitorService) initializeCollectors() error {
	for name, collectorCfg := range s.config.Collectors {
		if !collectorCfg.Enabled {
			s.logger.Debug("Collector is disabled, skipping", zap.String("collector", name))
			continue
		}

		collector, exists := s.collectorRegistry.Get(name)
		if !exists {
			s.logger.Error("Collector is enabled but not registered", zap.String("collector", name))
			continue
		}

		settings := collectorCfg.Settings
		if settings == nil {
			settings = make(map[string]interface{})
		}

		if err := collector.Init(settings); err != nil {
			s.logger.Error("Failed to initialize collector", zap.String("collector", name), zap.Error(err))
			return err
		}

		log.Printf("Collector %s initialized", name)
	}

	return nil
}

// initializeNotifiers initializes enabled notifiers
func (s *MonitorService) initializeNotifiers() error {
	// Initialize email notifier if enabled
	if s.config.Notifications.Email.Enabled {
		notifier, exists := s.notifierRegistry.Get("email")
		if !exists {
			return errors.New("email notifier is enabled but not registered")
		}

		emailCfg := s.config.Notifications.Email

		// Convert email config to map
		config := map[string]interface{}{
			"from":        emailCfg.From,
			"to":          emailCfg.To,
			"smtp_server": emailCfg.SMTPServer,
			"smtp_port":   emailCfg.SMTPPort,
			"username":    emailCfg.Username,
			"password":    emailCfg.Password,
		}

		if err := notifier.Init(config); err != nil {
			s.logger.Error("Failed to initialize email notifier", zap.Error(err))
			return err
		}

		s.logger.Info("Email notifier initialized")
	}

	return nil
}

// startCollectorTasks starts all enabled collector tasks
func (s *MonitorService) startCollectorTasks() error {
	for name, collectorCfg := range s.config.Collectors {
		if !collectorCfg.Enabled {
			continue
		}

		collector, exists := s.collectorRegistry.Get(name)
		if !exists {
			continue
		}

		interval := s.config.GetCollectorInterval(name)
		if interval <= 0 {
			err := fmt.Errorf("invalid interval for collector %s", name)
			s.logger.Error("Invalid collector interval", zap.String("collector", name), zap.Error(err))
			return err
		}

		// Start collector task
		if err := s.startCollectorTask(collector, interval); err != nil {
			err := fmt.Errorf("failed to start collector task %s: %w", name, err)
			s.logger.Error("Failed to start collector task", zap.String("collector", name), zap.Error(err))
			return err
		}

		s.logger.Info("Collector task started", zap.String("collector", name), zap.Duration("interval", interval))
	}

	return nil
}

// startCollectorTask starts a collector task with the specified interval
func (s *MonitorService) startCollectorTask(collector collectors.Collector, interval time.Duration) error {
	taskCtx, cancel := context.WithCancel(s.ctx)

	s.mu.Lock()
	s.collectorTasks[collector.Name()] = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run immediately on start
		if err := s.runCollector(taskCtx, collector); err != nil {
			log.Printf("Error collecting metrics for %s: %v", collector.Name(), err)
		}

		for {
			select {
			case <-taskCtx.Done():
				log.Printf("Collector task %s stopping", collector.Name())
				return
			case <-ticker.C:
				if err := s.runCollector(taskCtx, collector); err != nil {
					log.Printf("Error collecting metrics for %s: %v", collector.Name(), err)
				}
			}
		}
	}()

	return nil
}

// runCollector executes a collector and processes its results
func (s *MonitorService) runCollector(ctx context.Context, collector collectors.Collector) error {
	// Create a timeout context for the collection operation
	collectionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Collect metrics
	results, err := collector.Collect(collectionCtx)
	if err != nil {
		s.logger.Error("Failed to collect metrics", zap.String("collector", collector.Name()), zap.Error(err))
		return err
	}

	// Process results
	return s.processResults(ctx, results)
}

// processResults processes collector results and sends notifications if needed
func (s *MonitorService) processResults(ctx context.Context, results []collectors.Result) error {
	// Check if there are any unhealthy results
	var unhealthyResults []collectors.Result
	for _, result := range results {
		if !result.IsHealthy {
			unhealthyResults = append(unhealthyResults, result)
			log.Printf("Unhealthy result from %s: %s", result.Collector, result.Message)
		}
	}

	// If no unhealthy results, nothing to do
	if len(unhealthyResults) == 0 {
		return nil
	}

	// Send notifications
	return s.sendNotifications(ctx, unhealthyResults)
}

// sendNotifications sends notifications for unhealthy results
func (s *MonitorService) sendNotifications(ctx context.Context, results []collectors.Result) error {
	// Create a timeout context for notification operations
	notifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Send to all enabled notifiers
	var errs []error

	// Check if email notifications are enabled
	if s.config.Notifications.Email.Enabled {
		notifier, exists := s.notifierRegistry.Get("email")
		if exists {
			if err := notifier.Notify(notifyCtx, results); err != nil {
				errs = append(errs, fmt.Errorf("email notification failed: %w", err))
			} else {
				log.Printf("Email notification sent for %d issues", len(results))
			}
		}
	}

	// Check if there were any errors
	if len(errs) > 0 {
		errStrings := make([]string, len(errs))
		for i, err := range errs {
			errStrings[i] = err.Error()
		}
		s.logger.Error("Notification errors", zap.Strings("errors", errStrings))
		return fmt.Errorf("notification errors: %v", errStrings)
	}

	return nil
}
