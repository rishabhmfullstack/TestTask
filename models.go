package main

import (
	"sync"
	"time"
)

// JobStatus represents the status of a file processing job
type JobStatus string

const (
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ProcessingJob represents a file processing job
type ProcessingJob struct {
	ID        string    `json:"id"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	FilePath  string    `json:"file_path,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// UploadResponse represents the response for upload endpoint
type UploadResponse struct {
	ID string `json:"id"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// JobStore manages in-memory storage of processing jobs
type JobStore struct {
	jobs map[string]*ProcessingJob
	mu   sync.RWMutex
}

// NewJobStore creates a new job store
func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*ProcessingJob),
	}
}

// CreateJob creates a new processing job
func (js *JobStore) CreateJob(id string) *ProcessingJob {
	js.mu.Lock()
	defer js.mu.Unlock()

	job := &ProcessingJob{
		ID:        id,
		Status:    JobStatusProcessing,
		CreatedAt: time.Now(),
	}
	js.jobs[id] = job
	return job
}

// GetJob retrieves a job by ID
func (js *JobStore) GetJob(id string) (*ProcessingJob, bool) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	job, exists := js.jobs[id]
	return job, exists
}

// UpdateJobStatus updates the status of a job
func (js *JobStore) UpdateJobStatus(id string, status JobStatus, filePath string, errorMsg string) {
	js.mu.Lock()
	defer js.mu.Unlock()

	if job, exists := js.jobs[id]; exists {
		job.Status = status
		if filePath != "" {
			job.FilePath = filePath
		}
		if errorMsg != "" {
			job.Error = errorMsg
		}
	}
}
