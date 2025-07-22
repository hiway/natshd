package service

import (
	"encoding/json"
	"testing"
)

func TestServiceDefinition_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectedDef ServiceDefinition
		expectError bool
	}{
		{
			name: "valid service definition",
			jsonData: `{
				"name": "HelloService",
				"version": "1.0.0",
				"description": "A simple hello world service",
				"endpoints": [
					{
						"name": "SayHello",
						"subject": "hello.world"
					}
				]
			}`,
			expectedDef: ServiceDefinition{
				Name:        "HelloService",
				Version:     "1.0.0",
				Description: "A simple hello world service",
				Endpoints: []Endpoint{
					{
						Name:    "SayHello",
						Subject: "hello.world",
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple endpoints",
			jsonData: `{
				"name": "UserService",
				"version": "2.1.0",
				"description": "User management service",
				"endpoints": [
					{
						"name": "GetUser",
						"subject": "user.get"
					},
					{
						"name": "CreateUser",
						"subject": "user.create"
					},
					{
						"name": "DeleteUser",
						"subject": "user.delete"
					}
				]
			}`,
			expectedDef: ServiceDefinition{
				Name:        "UserService",
				Version:     "2.1.0",
				Description: "User management service",
				Endpoints: []Endpoint{
					{Name: "GetUser", Subject: "user.get"},
					{Name: "CreateUser", Subject: "user.create"},
					{Name: "DeleteUser", Subject: "user.delete"},
				},
			},
			expectError: false,
		},
		{
			name: "minimal valid definition",
			jsonData: `{
				"name": "MinimalService",
				"endpoints": [
					{
						"name": "DoSomething",
						"subject": "minimal.do"
					}
				]
			}`,
			expectedDef: ServiceDefinition{
				Name:        "MinimalService",
				Version:     "",
				Description: "",
				Endpoints: []Endpoint{
					{Name: "DoSomething", Subject: "minimal.do"},
				},
			},
			expectError: false,
		},
		{
			name: "invalid JSON",
			jsonData: `{
				"name": "BadService"
				"endpoints": [
			`,
			expectedDef: ServiceDefinition{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var def ServiceDefinition
			err := json.Unmarshal([]byte(tt.jsonData), &def)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if def.Name != tt.expectedDef.Name {
				t.Errorf("Expected Name %s, got %s", tt.expectedDef.Name, def.Name)
			}

			if def.Version != tt.expectedDef.Version {
				t.Errorf("Expected Version %s, got %s", tt.expectedDef.Version, def.Version)
			}

			if def.Description != tt.expectedDef.Description {
				t.Errorf("Expected Description %s, got %s", tt.expectedDef.Description, def.Description)
			}

			if len(def.Endpoints) != len(tt.expectedDef.Endpoints) {
				t.Errorf("Expected %d endpoints, got %d", len(tt.expectedDef.Endpoints), len(def.Endpoints))
			}

			for i, endpoint := range def.Endpoints {
				if i >= len(tt.expectedDef.Endpoints) {
					break
				}
				expected := tt.expectedDef.Endpoints[i]
				if endpoint.Name != expected.Name {
					t.Errorf("Endpoint %d: Expected Name %s, got %s", i, expected.Name, endpoint.Name)
				}
				if endpoint.Subject != expected.Subject {
					t.Errorf("Endpoint %d: Expected Subject %s, got %s", i, expected.Subject, endpoint.Subject)
				}
			}
		})
	}
}

func TestServiceDefinition_Validate(t *testing.T) {
	tests := []struct {
		name        string
		def         ServiceDefinition
		expectError bool
	}{
		{
			name: "valid definition",
			def: ServiceDefinition{
				Name:        "ValidService",
				Version:     "1.0.0",
				Description: "A valid service",
				Endpoints: []Endpoint{
					{Name: "DoSomething", Subject: "valid.do"},
				},
			},
			expectError: false,
		},
		{
			name: "empty name",
			def: ServiceDefinition{
				Name: "",
				Endpoints: []Endpoint{
					{Name: "DoSomething", Subject: "test.do"},
				},
			},
			expectError: true,
		},
		{
			name: "no endpoints",
			def: ServiceDefinition{
				Name:      "NoEndpoints",
				Endpoints: []Endpoint{},
			},
			expectError: true,
		},
		{
			name: "nil endpoints",
			def: ServiceDefinition{
				Name:      "NilEndpoints",
				Endpoints: nil,
			},
			expectError: true,
		},
		{
			name: "endpoint with empty name",
			def: ServiceDefinition{
				Name: "BadEndpoint",
				Endpoints: []Endpoint{
					{Name: "", Subject: "bad.endpoint"},
				},
			},
			expectError: true,
		},
		{
			name: "endpoint with empty subject",
			def: ServiceDefinition{
				Name: "BadEndpoint",
				Endpoints: []Endpoint{
					{Name: "BadEndpoint", Subject: ""},
				},
			},
			expectError: true,
		},
		{
			name: "duplicate endpoint subjects",
			def: ServiceDefinition{
				Name: "DuplicateSubjects",
				Endpoints: []Endpoint{
					{Name: "First", Subject: "duplicate.subject"},
					{Name: "Second", Subject: "duplicate.subject"},
				},
			},
			expectError: true,
		},
		{
			name: "duplicate endpoint names",
			def: ServiceDefinition{
				Name: "DuplicateNames",
				Endpoints: []Endpoint{
					{Name: "Duplicate", Subject: "first.subject"},
					{Name: "Duplicate", Subject: "second.subject"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.def.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestEndpoint_Validate(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    Endpoint
		expectError bool
	}{
		{
			name: "valid endpoint",
			endpoint: Endpoint{
				Name:    "ValidEndpoint",
				Subject: "valid.subject",
			},
			expectError: false,
		},
		{
			name: "empty name",
			endpoint: Endpoint{
				Name:    "",
				Subject: "valid.subject",
			},
			expectError: true,
		},
		{
			name: "empty subject",
			endpoint: Endpoint{
				Name:    "ValidName",
				Subject: "",
			},
			expectError: true,
		},
		{
			name: "subject with spaces",
			endpoint: Endpoint{
				Name:    "ValidName",
				Subject: "invalid subject",
			},
			expectError: true,
		},
		{
			name: "subject with special chars",
			endpoint: Endpoint{
				Name:    "ValidName",
				Subject: "invalid.subject!",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.endpoint.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}
