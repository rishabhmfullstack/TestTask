package main

import (
	"fmt"
	"testing"
	"time"
)

func TestNewJobStore(t *testing.T) {
	store := NewJobStore()
	if store == nil {
		t.Fatal("NewJobStore() returned nil")
	}
	if store.jobs == nil {
		t.Fatal("Jobs map is nil")
	}
	if len(store.jobs) != 0 {
		t.Errorf("Expected empty jobs map, got %d jobs", len(store.jobs))
	}
}

func TestCreateJob(t *testing.T) {
	store := NewJobStore()
	jobID := "test-job-123"

	job := store.CreateJob(jobID)
	if job == nil {
		t.Fatal("CreateJob returned nil")
	}

	// Verify job properties
	if job.ID != jobID {
		t.Errorf("Job ID mismatch. Expected: %s, Got: %s", jobID, job.ID)
	}
	if job.Status != JobStatusProcessing {
		t.Errorf("Job status mismatch. Expected: %s, Got: %s", JobStatusProcessing, job.Status)
	}
	if job.CreatedAt.IsZero() {
		t.Error("Job CreatedAt is zero")
	}
	if job.FilePath != "" {
		t.Errorf("Job FilePath should be empty, got: %s", job.FilePath)
	}
	if job.Error != "" {
		t.Errorf("Job Error should be empty, got: %s", job.Error)
	}

	// Verify job is stored
	retrievedJob, exists := store.GetJob(jobID)
	if !exists {
		t.Error("Job was not stored")
	}
	if retrievedJob.ID != jobID {
		t.Errorf("Retrieved job ID mismatch. Expected: %s, Got: %s", jobID, retrievedJob.ID)
	}
}

func TestGetJob(t *testing.T) {
	store := NewJobStore()
	jobID := "test-job-123"

	// Test getting non-existent job
	job, exists := store.GetJob(jobID)
	if exists {
		t.Error("Expected non-existent job to return false")
	}
	if job != nil {
		t.Error("Expected non-existent job to return nil")
	}

	// Create and test getting existing job
	createdJob := store.CreateJob(jobID)
	retrievedJob, exists := store.GetJob(jobID)
	if !exists {
		t.Error("Expected existing job to return true")
	}
	if retrievedJob == nil {
		t.Error("Expected existing job to return non-nil")
	}
	if retrievedJob.ID != createdJob.ID {
		t.Errorf("Retrieved job ID mismatch. Expected: %s, Got: %s", createdJob.ID, retrievedJob.ID)
	}
}

func TestUpdateJobStatus(t *testing.T) {
	store := NewJobStore()
	jobID := "test-job-123"

	// Create job
	store.CreateJob(jobID)

	// Test updating to completed status
	filePath := "/path/to/processed/file.csv"
	store.UpdateJobStatus(jobID, JobStatusCompleted, filePath, "")

	job, exists := store.GetJob(jobID)
	if !exists {
		t.Fatal("Job should exist")
	}
	if job.Status != JobStatusCompleted {
		t.Errorf("Job status mismatch. Expected: %s, Got: %s", JobStatusCompleted, job.Status)
	}
	if job.FilePath != filePath {
		t.Errorf("Job file path mismatch. Expected: %s, Got: %s", filePath, job.FilePath)
	}
	if job.Error != "" {
		t.Errorf("Job error should be empty, got: %s", job.Error)
	}

	// Test updating to failed status
	errorMsg := "Processing failed"
	store.UpdateJobStatus(jobID, JobStatusFailed, "", errorMsg)

	job, exists = store.GetJob(jobID)
	if !exists {
		t.Fatal("Job should exist")
	}
	if job.Status != JobStatusFailed {
		t.Errorf("Job status mismatch. Expected: %s, Got: %s", JobStatusFailed, job.Status)
	}
	if job.Error != errorMsg {
		t.Errorf("Job error mismatch. Expected: %s, Got: %s", errorMsg, job.Error)
	}

	// Test updating non-existent job (should not panic)
	store.UpdateJobStatus("non-existent-job", JobStatusCompleted, "", "")
}

func TestJobStatusConstants(t *testing.T) {
	// Test that constants have expected values
	if JobStatusProcessing != "processing" {
		t.Errorf("JobStatusProcessing mismatch. Expected: processing, Got: %s", JobStatusProcessing)
	}
	if JobStatusCompleted != "completed" {
		t.Errorf("JobStatusCompleted mismatch. Expected: completed, Got: %s", JobStatusCompleted)
	}
	if JobStatusFailed != "failed" {
		t.Errorf("JobStatusFailed mismatch. Expected: failed, Got: %s", JobStatusFailed)
	}
}

func TestProcessingJobStruct(t *testing.T) {
	now := time.Now()
	job := &ProcessingJob{
		ID:        "test-id",
		Status:    JobStatusProcessing,
		CreatedAt: now,
		FilePath:  "/test/path.csv",
		Error:     "test error",
	}

	// Test struct fields
	if job.ID != "test-id" {
		t.Errorf("ID mismatch. Expected: test-id, Got: %s", job.ID)
	}
	if job.Status != JobStatusProcessing {
		t.Errorf("Status mismatch. Expected: %s, Got: %s", JobStatusProcessing, job.Status)
	}
	if !job.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch. Expected: %v, Got: %v", now, job.CreatedAt)
	}
	if job.FilePath != "/test/path.csv" {
		t.Errorf("FilePath mismatch. Expected: /test/path.csv, Got: %s", job.FilePath)
	}
	if job.Error != "test error" {
		t.Errorf("Error mismatch. Expected: test error, Got: %s", job.Error)
	}
}

func TestUploadResponseStruct(t *testing.T) {
	response := UploadResponse{ID: "test-id"}
	if response.ID != "test-id" {
		t.Errorf("UploadResponse ID mismatch. Expected: test-id, Got: %s", response.ID)
	}
}

func TestErrorResponseStruct(t *testing.T) {
	response := ErrorResponse{Error: "test error"}
	if response.Error != "test error" {
		t.Errorf("ErrorResponse Error mismatch. Expected: test error, Got: %s", response.Error)
	}
}

func TestJobStoreConcurrency(t *testing.T) {
	store := NewJobStore()

	// Test concurrent job creation
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			jobID := fmt.Sprintf("job-%d", index)
			job := store.CreateJob(jobID)

			// Verify job was created correctly
			if job.ID != jobID {
				t.Errorf("Concurrent job creation failed for job %d", index)
			}

			// Update job status
			store.UpdateJobStatus(jobID, JobStatusCompleted, "/test/path", "")

			// Retrieve job
			retrievedJob, exists := store.GetJob(jobID)
			if !exists || retrievedJob.Status != JobStatusCompleted {
				t.Errorf("Concurrent job retrieval failed for job %d", index)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all jobs were created
	if len(store.jobs) != 10 {
		t.Errorf("Expected 10 jobs, got %d", len(store.jobs))
	}
}

func TestJobStoreMultipleOperations(t *testing.T) {
	store := NewJobStore()

	// Create multiple jobs
	jobIDs := []string{"job1", "job2", "job3", "job4", "job5"}

	for _, jobID := range jobIDs {
		store.CreateJob(jobID)
	}

	// Verify all jobs exist
	for _, jobID := range jobIDs {
		job, exists := store.GetJob(jobID)
		if !exists {
			t.Errorf("Job %s should exist", jobID)
		}
		if job.ID != jobID {
			t.Errorf("Job ID mismatch for %s", jobID)
		}
	}

	// Update some jobs to completed
	completedJobs := []string{"job1", "job3", "job5"}
	for _, jobID := range completedJobs {
		store.UpdateJobStatus(jobID, JobStatusCompleted, "/path/"+jobID+".csv", "")
	}

	// Update some jobs to failed
	failedJobs := []string{"job2", "job4"}
	for _, jobID := range failedJobs {
		store.UpdateJobStatus(jobID, JobStatusFailed, "", "Error processing "+jobID)
	}

	// Verify status updates
	for _, jobID := range completedJobs {
		job, _ := store.GetJob(jobID)
		if job.Status != JobStatusCompleted {
			t.Errorf("Job %s should be completed", jobID)
		}
	}

	for _, jobID := range failedJobs {
		job, _ := store.GetJob(jobID)
		if job.Status != JobStatusFailed {
			t.Errorf("Job %s should be failed", jobID)
		}
		if job.Error == "" {
			t.Errorf("Job %s should have error message", jobID)
		}
	}
}
