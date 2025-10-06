package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// App represents the main application
type App struct {
	jobStore     *JobStore
	csvProcessor *CSVProcessor
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{
		jobStore:     NewJobStore(),
		csvProcessor: NewCSVProcessor(),
	}
}

// UploadHandler handles file upload requests
func (app *App) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		app.sendErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Get the file from form data
	file, handler, err := r.FormFile("file")
	if err != nil {
		app.sendErrorResponse(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Validate file type
	contentType := handler.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") && !strings.HasSuffix(strings.ToLower(handler.Filename), ".csv") {
		app.sendErrorResponse(w, http.StatusBadRequest, "File must be a CSV file")
		return
	}

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		app.sendErrorResponse(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Generate unique job ID
	jobID := uuid.New().String()

	// Create job
	app.jobStore.CreateJob(jobID)

	// Process file asynchronously
	go app.processFileAsync(jobID, fileData, handler.Filename)

	// Send response with job ID
	response := UploadResponse{ID: jobID}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DownloadHandler handles file download requests
func (app *App) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	// Get job from store
	job, exists := app.jobStore.GetJob(jobID)
	if !exists {
		app.sendErrorResponse(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	// Check job status
	switch job.Status {
	case JobStatusProcessing:
		w.WriteHeader(http.StatusLocked) // 423
		return
	case JobStatusFailed:
		app.sendErrorResponse(w, http.StatusInternalServerError, job.Error)
		return
	case JobStatusCompleted:
		// Serve the processed file
		app.serveFile(w, job.FilePath)
		return
	default:
		app.sendErrorResponse(w, http.StatusInternalServerError, "Unknown job status")
		return
	}
}

// processFileAsync processes the uploaded file asynchronously
func (app *App) processFileAsync(jobID string, fileData []byte, filename string) {
	// Save uploaded file
	uploadPath, err := app.csvProcessor.SaveUploadedFile(fileData, fmt.Sprintf("upload_%s_%s", jobID, filename))
	if err != nil {
		app.jobStore.UpdateJobStatus(jobID, JobStatusFailed, "", fmt.Sprintf("Failed to save uploaded file: %v", err))
		return
	}

	// Generate processed file path
	processedPath := app.csvProcessor.GetProcessedFilePath(jobID)

	// Process CSV file
	err = app.csvProcessor.ProcessCSV(uploadPath, processedPath)
	if err != nil {
		app.jobStore.UpdateJobStatus(jobID, JobStatusFailed, "", fmt.Sprintf("Failed to process CSV: %v", err))
		return
	}

	// Update job status to completed
	app.jobStore.UpdateJobStatus(jobID, JobStatusCompleted, processedPath, "")
}

// serveFile serves a file as a blob
func (app *App) serveFile(w http.ResponseWriter, filePath string) {
	// Set appropriate headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filePath)))

	// Open and serve file
	file, err := os.Open(filePath)
	if err != nil {
		app.sendErrorResponse(w, http.StatusInternalServerError, "Failed to open processed file")
		return
	}
	defer file.Close()

	// Copy file to response
	_, err = io.Copy(w, file)
	if err != nil {
		app.sendErrorResponse(w, http.StatusInternalServerError, "Failed to serve file")
		return
	}
}

// sendErrorResponse sends an error response
func (app *App) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	response := ErrorResponse{Error: message}
	json.NewEncoder(w).Encode(response)
}
