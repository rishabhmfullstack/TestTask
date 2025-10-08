package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCSVProcessor(t *testing.T) {
	processor := NewCSVProcessor()
	if processor == nil {
		t.Fatal("NewCSVProcessor() returned nil")
	}
	if processor.validator == nil {
		t.Fatal("Email validator is nil")
	}
}

func TestProcessCSV(t *testing.T) {
	processor := NewCSVProcessor()

	// Create temporary test files
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Test data
	testCSV := `name,email,phone,company
John Doe,john.doe@example.com,555-1234,Acme Corp
Jane Smith,jane.smith@company.org,555-5678,Tech Inc
Bob Johnson,bob@invalid-email,555-9012,Startup LLC
Alice Brown,alice.brown@domain.co.uk,555-3456,Global Ltd
Charlie Wilson,charlie@test.com,555-7890,Local Business
David Lee,david.lee@email.net,555-2468,Enterprise Corp`

	// Write test CSV
	err := os.WriteFile(inputFile, []byte(testCSV), 0644)
	if err != nil {
		t.Fatalf("Failed to write test CSV: %v", err)
	}

	// Process CSV
	err = processor.ProcessCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ProcessCSV failed: %v", err)
	}

	// Read and verify output
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputLines := strings.Split(strings.TrimSpace(string(outputData)), "\n")
	expectedLines := 7 // 1 header + 6 data rows

	if len(outputLines) != expectedLines {
		t.Fatalf("Expected %d lines, got %d", expectedLines, len(outputLines))
	}

	// Check header
	expectedHeader := "name,email,phone,company,has_email"
	if outputLines[0] != expectedHeader {
		t.Errorf("Header mismatch. Expected: %s, Got: %s", expectedHeader, outputLines[0])
	}

	// Check data rows
	expectedResults := []bool{true, true, false, true, true, true} // has_email values
	for i, line := range outputLines[1:] {
		fields := strings.Split(line, ",")
		if len(fields) != 5 {
			t.Errorf("Row %d has %d fields, expected 5", i+1, len(fields))
			continue
		}

		hasEmailStr := fields[4]
		expected := expectedResults[i]
		actual := hasEmailStr == "true"

		if actual != expected {
			t.Errorf("Row %d: has_email = %s, expected %t", i+1, hasEmailStr, expected)
		}
	}
}

func TestProcessCSVWithEmptyRows(t *testing.T) {
	processor := NewCSVProcessor()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Test CSV with empty rows
	testCSV := `name,email,phone,company
John Doe,john.doe@example.com,555-1234,Acme Corp

Jane Smith,jane.smith@company.org,555-5678,Tech Inc

Bob Johnson,bob@invalid-email,555-9012,Startup LLC`

	err := os.WriteFile(inputFile, []byte(testCSV), 0644)
	if err != nil {
		t.Fatalf("Failed to write test CSV: %v", err)
	}

	err = processor.ProcessCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ProcessCSV failed: %v", err)
	}

	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputLines := strings.Split(strings.TrimSpace(string(outputData)), "\n")
	expectedLines := 4 // 1 header + 3 data rows (empty rows should be skipped)

	if len(outputLines) != expectedLines {
		t.Fatalf("Expected %d lines, got %d", expectedLines, len(outputLines))
	}
}

func TestProcessCSVWithOnlyHeader(t *testing.T) {
	processor := NewCSVProcessor()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Test CSV with only header
	testCSV := `name,email,phone,company`

	err := os.WriteFile(inputFile, []byte(testCSV), 0644)
	if err != nil {
		t.Fatalf("Failed to write test CSV: %v", err)
	}

	err = processor.ProcessCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ProcessCSV failed: %v", err)
	}

	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputLines := strings.Split(strings.TrimSpace(string(outputData)), "\n")
	expectedLines := 1 // Only header

	if len(outputLines) != expectedLines {
		t.Fatalf("Expected %d lines, got %d", expectedLines, len(outputLines))
	}

	expectedHeader := "name,email,phone,company,has_email"
	if outputLines[0] != expectedHeader {
		t.Errorf("Header mismatch. Expected: %s, Got: %s", expectedHeader, outputLines[0])
	}
}

func TestProcessCSVWithEmptyFile(t *testing.T) {
	processor := NewCSVProcessor()

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	outputFile := filepath.Join(tempDir, "output.csv")

	// Create empty file
	err := os.WriteFile(inputFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test CSV: %v", err)
	}

	err = processor.ProcessCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ProcessCSV failed: %v", err)
	}

	// Empty file should result in empty output
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(strings.TrimSpace(string(outputData))) != 0 {
		t.Errorf("Expected empty output for empty input, got: %s", string(outputData))
	}
}

func TestProcessCSVErrorHandling(t *testing.T) {
	processor := NewCSVProcessor()

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Test with non-existent input file
	err := processor.ProcessCSV("non-existent-file.csv", outputFile)
	if err == nil {
		t.Error("Expected error for non-existent input file")
	}

	// Test with invalid output directory
	err = processor.ProcessCSV("", "/invalid/path/output.csv")
	if err == nil {
		t.Error("Expected error for invalid output path")
	}
}

func TestSaveUploadedFile(t *testing.T) {
	processor := NewCSVProcessor()

	tempDir := t.TempDir()

	// Change to temp directory to avoid creating uploads in project root
	originalDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalDir)

	testData := []byte("test,data,here\n1,2,3")
	filename := "test.csv"

	filePath, err := processor.SaveUploadedFile(testData, filename)
	if err != nil {
		t.Fatalf("SaveUploadedFile failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File was not created at %s", filePath)
	}

	// Verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(content) != string(testData) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", string(testData), string(content))
	}

	// Verify file path format
	expectedPath := filepath.Join("uploads", filename)
	if filePath != expectedPath {
		t.Errorf("File path mismatch. Expected: %s, Got: %s", expectedPath, filePath)
	}
}

func TestGetProcessedFilePath(t *testing.T) {
	processor := NewCSVProcessor()

	jobID := "test-job-id"
	expectedPath := filepath.Join("uploads", "processed_test-job-id.csv")

	actualPath := processor.GetProcessedFilePath(jobID)
	if actualPath != expectedPath {
		t.Errorf("GetProcessedFilePath mismatch. Expected: %s, Got: %s", expectedPath, actualPath)
	}
}

func TestCSVProcessorConcurrency(t *testing.T) {
	processor := NewCSVProcessor()

	// Test concurrent processing
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(index int) {
			defer func() { done <- true }()

			tempDir := t.TempDir()
			inputFile := filepath.Join(tempDir, "input.csv")
			outputFile := filepath.Join(tempDir, "output.csv")

			testCSV := `name,email
Test User,test@example.com`

			os.WriteFile(inputFile, []byte(testCSV), 0644)
			processor.ProcessCSV(inputFile, outputFile)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}
