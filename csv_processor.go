package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CSVProcessor handles CSV file processing
type CSVProcessor struct {
	validator *EmailValidator
}

// NewCSVProcessor creates a new CSV processor
func NewCSVProcessor() *CSVProcessor {
	return &CSVProcessor{
		validator: NewEmailValidator(),
	}
}

// ProcessCSV processes a CSV file and adds email validation column
func (cp *CSVProcessor) ProcessCSV(inputPath, outputPath string) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create CSV reader and writer
	reader := csv.NewReader(inputFile)
	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// Process each row
	rowNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}

		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// For header row (first row), add "has_email" column
		if rowNum == 0 {
			record = append(record, "has_email")
		} else {
			// For data rows, check if any field contains a valid email
			hasEmail := cp.validator.HasValidEmail(record)
			record = append(record, fmt.Sprintf("%t", hasEmail))
		}

		// Write the modified record
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row %d: %w", rowNum, err)
		}

		rowNum++
	}

	return nil
}

// SaveUploadedFile saves the uploaded file to the filesystem
func (cp *CSVProcessor) SaveUploadedFile(fileData []byte, filename string) (string, error) {
	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Generate file path
	filePath := filepath.Join(uploadsDir, filename)

	// Write file
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return "", fmt.Errorf("failed to write uploaded file: %w", err)
	}

	return filePath, nil
}

// GetProcessedFilePath returns the path for the processed file
func (cp *CSVProcessor) GetProcessedFilePath(jobID string) string {
	return filepath.Join("uploads", fmt.Sprintf("processed_%s.csv", jobID))
}
