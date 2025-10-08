package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestMainIntegration(t *testing.T) {
	// Create app instance
	app := NewApp()
	router := mux.NewRouter()

	// Setup routes (same as main.go)
	api := router.PathPrefix("/API").Subrouter()
	api.HandleFunc("/upload", app.UploadHandler).Methods("POST")
	api.HandleFunc("/download/{id}", app.DownloadHandler).Methods("GET")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

	// Test health endpoint
	t.Run("Health Check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check failed. Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "OK" {
			t.Errorf("Health check response mismatch. Expected 'OK', got '%s'", w.Body.String())
		}
	})

	// Test complete upload and download flow
	t.Run("Complete Upload and Download Flow", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Create uploads directory
		os.MkdirAll("uploads", 0755)

		// Test CSV content
		csvContent := `name,email,phone,company
John Doe,john.doe@example.com,555-1234,Acme Corp
Jane Smith,jane.smith@company.org,555-5678,Tech Inc
Bob Johnson,bob@invalid-email,555-9012,Startup LLC
Alice Brown,alice.brown@domain.co.uk,555-3456,Global Ltd`

		// Step 1: Upload CSV file
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte(csvContent))
		writer.Close()

		req := httptest.NewRequest("POST", "/API/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Upload failed. Expected status 200, got %d", w.Code)
		}

		var uploadResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &uploadResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal upload response: %v", err)
		}

		jobID, exists := uploadResponse["id"]
		if !exists {
			t.Fatal("Upload response missing job ID")
		}

		// Step 2: Wait for processing to complete
		time.Sleep(200 * time.Millisecond)

		// Step 3: Download processed file
		downloadReq := httptest.NewRequest("GET", fmt.Sprintf("/API/download/%s", jobID), nil)
		downloadW := httptest.NewRecorder()
		router.ServeHTTP(downloadW, downloadReq)

		if downloadW.Code != http.StatusOK {
			t.Errorf("Download failed. Expected status 200, got %d", downloadW.Code)
		}

		// Verify downloaded content
		downloadedContent := downloadW.Body.String()
		lines := strings.Split(strings.TrimSpace(downloadedContent), "\n")

		// Should have 5 lines (1 header + 4 data rows)
		if len(lines) != 5 {
			t.Errorf("Expected 5 lines in downloaded file, got %d", len(lines))
		}

		// Check header
		expectedHeader := "name,email,phone,company,has_email"
		if lines[0] != expectedHeader {
			t.Errorf("Header mismatch. Expected: %s, Got: %s", expectedHeader, lines[0])
		}

		// Check data rows
		expectedHasEmail := []bool{true, true, false, true}
		for i, line := range lines[1:] {
			fields := strings.Split(line, ",")
			if len(fields) != 5 {
				t.Errorf("Row %d has %d fields, expected 5", i+1, len(fields))
				continue
			}

			hasEmailStr := fields[4]
			expected := expectedHasEmail[i]
			actual := hasEmailStr == "true"

			if actual != expected {
				t.Errorf("Row %d: has_email = %s, expected %t", i+1, hasEmailStr, expected)
			}
		}
	})

	// Test download while processing
	t.Run("Download While Processing", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(originalDir)

		// Create uploads directory
		os.MkdirAll("uploads", 0755)

		// Upload file
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "test.csv")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte("name,email\nJohn Doe,john@example.com"))
		writer.Close()

		req := httptest.NewRequest("POST", "/API/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Upload failed. Expected status 200, got %d", w.Code)
		}

		var uploadResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &uploadResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal upload response: %v", err)
		}

		jobID := uploadResponse["id"].(string)

		// Immediately try to download (should be processing)
		downloadReq := httptest.NewRequest("GET", fmt.Sprintf("/API/download/%s", jobID), nil)
		downloadW := httptest.NewRecorder()
		router.ServeHTTP(downloadW, downloadReq)

		if downloadW.Code != http.StatusLocked {
			t.Errorf("Expected status 423 (Locked), got %d", downloadW.Code)
		}
	})

	// Test invalid job ID
	t.Run("Invalid Job ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/API/download/invalid-job-id", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if response["error"] != "Invalid job ID" {
			t.Errorf("Expected 'Invalid job ID' error, got %v", response["error"])
		}
	})

	// Test invalid file upload
	t.Run("Invalid File Upload", func(t *testing.T) {
		// Upload non-CSV file
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write([]byte("This is not a CSV file"))
		writer.Close()

		req := httptest.NewRequest("POST", "/API/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if !strings.Contains(response["error"].(string), "CSV file") {
			t.Errorf("Expected CSV file error, got %v", response["error"])
		}
	})
}

func TestMainRoutes(t *testing.T) {
	// Test that all routes are properly configured
	app := NewApp()
	router := mux.NewRouter()

	// Setup routes
	api := router.PathPrefix("/API").Subrouter()
	api.HandleFunc("/upload", app.UploadHandler).Methods("POST")
	api.HandleFunc("/download/{id}", app.DownloadHandler).Methods("GET")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

	// Test route matching
	tests := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/health", http.StatusOK},
		{"POST", "/API/upload", http.StatusBadRequest},          // No file provided
		{"GET", "/API/download/test-id", http.StatusBadRequest}, // Invalid job ID
		{"GET", "/invalid-path", http.StatusNotFound},
		{"POST", "/API/invalid-endpoint", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestMainConcurrency(t *testing.T) {
	// Test concurrent requests
	app := NewApp()
	router := mux.NewRouter()

	// Setup routes
	api := router.PathPrefix("/API").Subrouter()
	api.HandleFunc("/upload", app.UploadHandler).Methods("POST")
	api.HandleFunc("/download/{id}", app.DownloadHandler).Methods("GET")
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create uploads directory
	os.MkdirAll("uploads", 0755)

	// Test concurrent uploads
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			// Upload file
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", fmt.Sprintf("test%d.csv", index))
			if err != nil {
				t.Errorf("Failed to create form file: %v", err)
				return
			}
			part.Write([]byte(fmt.Sprintf("name,email\nUser %d,user%d@example.com", index, index)))
			writer.Close()

			req := httptest.NewRequest("POST", "/API/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Upload %d failed with status %d", index, w.Code)
				return
			}

			// Get job ID
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			jobID := response["id"].(string)

			// Wait for processing
			time.Sleep(100 * time.Millisecond)

			// Download file
			downloadReq := httptest.NewRequest("GET", fmt.Sprintf("/API/download/%s", jobID), nil)
			downloadW := httptest.NewRecorder()
			router.ServeHTTP(downloadW, downloadReq)

			if downloadW.Code != http.StatusOK {
				t.Errorf("Download %d failed with status %d", index, downloadW.Code)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMainErrorHandling(t *testing.T) {
	app := NewApp()
	router := mux.NewRouter()

	// Setup routes
	api := router.PathPrefix("/API").Subrouter()
	api.HandleFunc("/upload", app.UploadHandler).Methods("POST")
	api.HandleFunc("/download/{id}", app.DownloadHandler).Methods("GET")

	// Test various error conditions
	tests := []struct {
		name           string
		method         string
		path           string
		body           io.Reader
		headers        map[string]string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "No file in upload",
			method:         "POST",
			path:           "/API/upload",
			body:           strings.NewReader(""),
			headers:        map[string]string{"Content-Type": "multipart/form-data"},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Invalid multipart form",
			method:         "POST",
			path:           "/API/upload",
			body:           strings.NewReader("invalid data"),
			headers:        map[string]string{"Content-Type": "multipart/form-data"},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Non-existent job download",
			method:         "GET",
			path:           "/API/download/non-existent-job",
			body:           nil,
			headers:        nil,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}
				if _, exists := response["error"]; !exists {
					t.Error("Expected error in response")
				}
			}
		})
	}
}
