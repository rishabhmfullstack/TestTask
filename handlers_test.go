package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
	if app.jobStore == nil {
		t.Fatal("JobStore is nil")
	}
	if app.csvProcessor == nil {
		t.Fatal("CSVProcessor is nil")
	}
}

func TestUploadHandler(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name           string
		fileContent    string
		fileName       string
		contentType    string
		expectedStatus int
		expectJobID    bool
	}{
		{
			name:           "Valid CSV file",
			fileContent:    "name,email\nJohn Doe,john@example.com",
			fileName:       "test.csv",
			contentType:    "text/csv",
			expectedStatus: http.StatusOK,
			expectJobID:    true,
		},
		{
			name:           "CSV file with .csv extension",
			fileContent:    "name,email\nJane Smith,jane@example.com",
			fileName:       "data.csv",
			contentType:    "application/octet-stream",
			expectedStatus: http.StatusOK,
			expectJobID:    true,
		},
		{
			name:           "No file provided",
			fileContent:    "",
			fileName:       "",
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
			expectJobID:    false,
		},
		{
			name:           "Invalid file type",
			fileContent:    "some text content",
			fileName:       "test.txt",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
			expectJobID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			if tt.fileContent != "" {
				part, err := writer.CreateFormFile("file", tt.fileName)
				if err != nil {
					t.Fatalf("Failed to create form file: %v", err)
				}
				part.Write([]byte(tt.fileContent))
			}

			writer.Close()

			// Create request
			req := httptest.NewRequest("POST", "/API/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			// Call handler
			app.UploadHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectJobID {
				if _, exists := response["id"]; !exists {
					t.Error("Expected job ID in response")
				}
				if response["id"] == "" {
					t.Error("Job ID should not be empty")
				}
			} else {
				if _, exists := response["error"]; !exists {
					t.Error("Expected error in response")
				}
			}
		})
	}
}

func TestDownloadHandler(t *testing.T) {
	app := NewApp()

	// Create a test job
	jobID := "test-job-123"
	app.jobStore.CreateJob(jobID)

	// Create a temporary file for the completed job test
	tempFile := filepath.Join(t.TempDir(), "processed_file.csv")
	testContent := "name,email,has_email\nJohn Doe,john@example.com,true"
	err := os.WriteFile(tempFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name           string
		jobID          string
		jobStatus      JobStatus
		filePath       string
		errorMsg       string
		expectedStatus int
		expectFile     bool
	}{
		{
			name:           "Job still processing",
			jobID:          jobID,
			jobStatus:      JobStatusProcessing,
			filePath:       "",
			errorMsg:       "",
			expectedStatus: http.StatusLocked, // 423
			expectFile:     false,
		},
		{
			name:           "Job completed successfully",
			jobID:          jobID,
			jobStatus:      JobStatusCompleted,
			filePath:       tempFile,
			errorMsg:       "",
			expectedStatus: http.StatusOK,
			expectFile:     true,
		},
		{
			name:           "Job failed",
			jobID:          jobID,
			jobStatus:      JobStatusFailed,
			filePath:       "",
			errorMsg:       "Processing failed",
			expectedStatus: http.StatusInternalServerError,
			expectFile:     false,
		},
		{
			name:           "Invalid job ID",
			jobID:          "invalid-job-id",
			jobStatus:      "",
			filePath:       "",
			errorMsg:       "",
			expectedStatus: http.StatusBadRequest,
			expectFile:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update job status if needed
			if tt.jobID == jobID {
				app.jobStore.UpdateJobStatus(tt.jobID, tt.jobStatus, tt.filePath, tt.errorMsg)
			}

			// Create request with mux vars
			req := httptest.NewRequest("GET", fmt.Sprintf("/API/download/%s", tt.jobID), nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.jobID})
			w := httptest.NewRecorder()

			// Call handler
			app.DownloadHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response content
			if tt.expectFile {
				// Should have file headers
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/octet-stream" {
					t.Errorf("Expected Content-Type application/octet-stream, got %s", contentType)
				}
				contentDisposition := w.Header().Get("Content-Disposition")
				if !strings.Contains(contentDisposition, "attachment") {
					t.Errorf("Expected Content-Disposition with attachment, got %s", contentDisposition)
				}
			} else if tt.expectedStatus == http.StatusInternalServerError || tt.expectedStatus == http.StatusBadRequest {
				// Should have error response
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

func TestProcessFileAsync(t *testing.T) {
	app := NewApp()

	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	// Create uploads directory
	os.MkdirAll("uploads", 0755)

	jobID := "test-async-job"
	fileData := []byte("name,email\nJohn Doe,john@example.com")
	filename := "test.csv"

	// Create job
	app.jobStore.CreateJob(jobID)

	// Process file asynchronously
	app.processFileAsync(jobID, fileData, filename)

	// Wait a bit for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Check job status
	job, exists := app.jobStore.GetJob(jobID)
	if !exists {
		t.Fatal("Job should exist")
	}

	if job.Status != JobStatusCompleted {
		t.Errorf("Expected job status completed, got %s", job.Status)
	}

	if job.FilePath == "" {
		t.Error("Job should have file path")
	}

	// Verify processed file exists
	if _, err := os.Stat(job.FilePath); os.IsNotExist(err) {
		t.Error("Processed file should exist")
	}
}

func TestServeFile(t *testing.T) {
	app := NewApp()

	// Create temporary file
	tempFile := filepath.Join(t.TempDir(), "test.csv")
	testContent := "name,email,has_email\nJohn Doe,john@example.com,true"
	err := os.WriteFile(tempFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Call serveFile
	app.serveFile(w, tempFile)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Expected Content-Type application/octet-stream, got %s", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment") {
		t.Errorf("Expected Content-Disposition with attachment, got %s", contentDisposition)
	}

	// Check body content
	body := w.Body.String()
	if body != testContent {
		t.Errorf("Response body mismatch. Expected: %s, Got: %s", testContent, body)
	}
}

func TestServeFileError(t *testing.T) {
	app := NewApp()

	// Create response recorder
	w := httptest.NewRecorder()

	// Call serveFile with non-existent file
	app.serveFile(w, "non-existent-file.csv")

	// Check response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	// Check error response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if _, exists := response["error"]; !exists {
		t.Error("Expected error in response")
	}
}

func TestSendErrorResponse(t *testing.T) {
	app := NewApp()

	// Create response recorder
	w := httptest.NewRecorder()

	// Call sendErrorResponse
	app.sendErrorResponse(w, http.StatusBadRequest, "Test error message")

	// Check status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response["error"] != "Test error message" {
		t.Errorf("Expected error message 'Test error message', got %v", response["error"])
	}
}

func TestUploadHandlerLargeFile(t *testing.T) {
	app := NewApp()

	// Create a large CSV content (but still under 10MB limit)
	largeContent := "name,email\n"
	for i := 0; i < 1000; i++ {
		largeContent += fmt.Sprintf("User %d,user%d@example.com\n", i, i)
	}

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "large.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(largeContent))
	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/API/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Call handler
	app.UploadHandler(w, req)

	// Should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should return job ID
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if _, exists := response["id"]; !exists {
		t.Error("Expected job ID in response")
	}
}

func TestUploadHandlerInvalidMultipart(t *testing.T) {
	app := NewApp()

	// Create request with invalid multipart data
	req := httptest.NewRequest("POST", "/API/upload", strings.NewReader("invalid multipart data"))
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()

	// Call handler
	app.UploadHandler(w, req)

	// Should return error
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Check error response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if _, exists := response["error"]; !exists {
		t.Error("Expected error in response")
	}
}

func TestHandlersConcurrency(t *testing.T) {
	app := NewApp()

	// Test concurrent uploads
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			// Create multipart form
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", fmt.Sprintf("test%d.csv", index))
			if err != nil {
				t.Errorf("Failed to create form file: %v", err)
				return
			}
			part.Write([]byte(fmt.Sprintf("name,email\nUser %d,user%d@example.com", index, index)))
			writer.Close()

			// Create request
			req := httptest.NewRequest("POST", "/API/upload", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			// Call handler
			app.UploadHandler(w, req)

			// Should succeed
			if w.Code != http.StatusOK {
				t.Errorf("Upload %d failed with status %d", index, w.Code)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}
