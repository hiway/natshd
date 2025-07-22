package service

import (
	"fmt"
	"regexp"
	"strings"
)

// ServiceDefinition represents the JSON structure returned by scripts when called with "info"
type ServiceDefinition struct {
	Name        string     `json:"name"`
	Version     string     `json:"version,omitempty"`
	Description string     `json:"description,omitempty"`
	Endpoints   []Endpoint `json:"endpoints"`
}

// Endpoint represents a single NATS subject endpoint for a service
type Endpoint struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

// Validate checks if the service definition is valid
func (sd ServiceDefinition) Validate() error {
	if strings.TrimSpace(sd.Name) == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(sd.Endpoints) == 0 {
		return fmt.Errorf("service must have at least one endpoint")
	}

	// Check for duplicate endpoint names and subjects
	nameMap := make(map[string]bool)
	subjectMap := make(map[string]bool)

	for i, endpoint := range sd.Endpoints {
		if err := endpoint.Validate(); err != nil {
			return fmt.Errorf("endpoint %d is invalid: %w", i, err)
		}

		if nameMap[endpoint.Name] {
			return fmt.Errorf("duplicate endpoint name: %s", endpoint.Name)
		}
		nameMap[endpoint.Name] = true

		if subjectMap[endpoint.Subject] {
			return fmt.Errorf("duplicate endpoint subject: %s", endpoint.Subject)
		}
		subjectMap[endpoint.Subject] = true
	}

	return nil
}

// Validate checks if the endpoint is valid
func (e Endpoint) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("endpoint name cannot be empty")
	}

	if strings.TrimSpace(e.Subject) == "" {
		return fmt.Errorf("endpoint subject cannot be empty")
	}

	// NATS subjects should only contain alphanumeric characters, dots, dashes, and underscores
	// and cannot contain spaces or other special characters
	validSubject := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validSubject.MatchString(e.Subject) {
		return fmt.Errorf("endpoint subject '%s' contains invalid characters, only alphanumeric, dots, dashes, and underscores are allowed", e.Subject)
	}

	return nil
}
