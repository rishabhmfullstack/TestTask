# CSV Email Validator API

A Go-based REST API that processes CSV files and adds email validation columns.

## Features

- Upload CSV files via REST API
- Automatic email validation using regex
- Asynchronous file processing
- In-memory job tracking
- File system storage for processed files

## API Endpoints

### 1. Upload CSV File

- **Endpoint**: `POST /API/upload`
- **Content-Type**: `multipart/form-data`
- **Body**: Form data with `file` field containing CSV file
- **Response**:
  - Success (200): `{"id": "uuid"}`
  - Error (400): `{"error": "error message"}`

### 2. Download Processed File

- **Endpoint**: `GET /API/download/{id}`
- **Response**:
  - Success (200): File blob
  - Processing (423): Job still in progress
  - Invalid ID (400): `{"error": "Invalid job ID"}`

### 3. Health Check

- **Endpoint**: `GET /health`
- **Response**: `OK`

## How It Works

1. Upload a CSV file to `/API/upload`
2. The system processes the file asynchronously:
   - Parses each row (ignoring empty rows)
   - Validates email addresses using regex
   - Adds a `has_email` column with `true`/`false` values
3. Download the processed file using the returned job ID

## Email Validation

The system uses a simple regex pattern to validate email addresses:

- Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
- Checks all fields in each row
- Returns `true` if any field contains a valid email

## Running the Application

1. Install dependencies:

   ```bash
   go mod tidy
   ```

2. Run the server:

   ```bash
   go run .
   ```

3. The server will start on port 8080

## Example Usage

### Upload a CSV file:

```bash
curl -X POST -F "file=@sample.csv" http://localhost:8080/API/upload
```

### Download processed file:

```bash
curl -O http://localhost:8080/API/download/{job-id}
```

## File Structure

- `main.go` - Application entry point and server setup
- `models.go` - Data structures and in-memory storage
- `handlers.go` - HTTP request handlers
- `csv_processor.go` - CSV processing logic
- `email_validator.go` - Email validation utilities
- `uploads/` - Directory for storing uploaded and processed files
