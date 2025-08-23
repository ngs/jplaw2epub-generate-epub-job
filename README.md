# jplaw2epub-generate-epub-job

Cloud Run Job for generating EPUB files from Japanese law data asynchronously.

## Overview

This Cloud Run Job processes EPUB generation requests from the main [jplaw2epub-web-api](https://github.com/ngs/jplaw2epub-web-api) service. It runs as a command-line application to handle heavy EPUB generation processes asynchronously without timeout constraints.

## Architecture

```
Main API → Cloud Run Jobs API → Cloud Run Job (Container)
                                    (this repo)
                                         ↓
                                  Cloud Storage
                                   (EPUB files)
```

## Setup

### Prerequisites

- Google Cloud Project with billing enabled
- Cloud Run API enabled
- Cloud Storage API enabled
- Container Registry or Artifact Registry enabled
- Service account with appropriate permissions

### Command-line Arguments

- `--revision-id`: Law revision ID to convert (required)
- `--version`: App version for storage path (default: `v1.0.0`)
- `--bucket`: GCS bucket name (defaults to `EPUB_BUCKET_NAME` env var)
- `--verbose`: Enable verbose logging

### Environment Variables

- `EPUB_BUCKET_NAME`: Cloud Storage bucket name (default: `epub-storage`)

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Run locally:
```bash
export EPUB_BUCKET_NAME=epub-storage
go run main.go --revision-id=335M50000002060_20250601_507M60000002046
```

### Deployment

#### Option 1: Using the Deploy Script

```bash
./deploy-job.sh
```

This script will:
1. Build the Docker image
2. Push to Container Registry
3. Create/update the Cloud Run Job

#### Option 2: Manual Deployment

```bash
# Build and push Docker image
docker build -t gcr.io/YOUR_PROJECT_ID/epub-generator .
docker push gcr.io/YOUR_PROJECT_ID/epub-generator

# Create Cloud Run Job
gcloud run jobs create epub-generator \
  --image=gcr.io/YOUR_PROJECT_ID/epub-generator \
  --region=asia-northeast1 \
  --parallelism=5 \
  --task-timeout=3600 \
  --max-retries=2 \
  --memory=1Gi \
  --cpu=2 \
  --set-env-vars="EPUB_BUCKET_NAME=epub-storage" \
  --project=YOUR_PROJECT_ID
```

## Storage Setup

Run the setup script to create and configure the storage bucket:

```bash
./scripts/setup-storage.sh
```

This script will:
- Create a Cloud Storage bucket for EPUB files
- Set lifecycle rules (auto-delete after 30 days)
- Configure CORS settings

## Usage

### Execute Job Manually

```bash
gcloud run jobs execute epub-generator \
  --region=asia-northeast1 \
  --args="--revision-id=335M50000002060_20250601_507M60000002046"
```

### Programmatic Execution (from Main API)

The main API triggers the job using Cloud Run Jobs API:

```go
jobsClient.RunJob(&runpb.RunJobRequest{
    Name: "projects/PROJECT/locations/REGION/jobs/epub-generator",
    Overrides: &runpb.RunJobRequest_Overrides{
        ContainerOverrides: [{
            Args: []string{
                "--revision-id", revisionID,
                "--version", version,
            },
        }],
    },
})
```

## File Structure in Cloud Storage

```
epub-storage/
├── v1.0.0/
│   ├── {id}.epub        # Generated EPUB file
│   └── {id}.status      # Processing status (temporary)
```

## Status File Format

```json
{
  "status": "PROCESSING|FAILED",
  "updatedAt": "2024-01-01T00:00:00Z",
  "error": "Error message if failed"
}
```

## Performance

- Timeout: 1 hour (3600 seconds)
- Memory: 1GB
- CPU: 2 cores
- Parallelism: 5 concurrent executions
- Max retries: 2
- Expected processing time: 30-60 seconds per EPUB

## Cost Estimation

For 1000 EPUB generations per month (assuming 1 minute per job):
- Cloud Run Jobs: ~$0.40 (CPU: $0.32 + Memory: $0.08)
- Cloud Storage: ~$0.07
- Total: ~$0.47/month

## Monitoring

View job executions:
```bash
gcloud run jobs executions list --job=epub-generator --region=asia-northeast1
```

View logs for a specific execution:
```bash
gcloud run jobs executions logs EXECUTION_ID --job=epub-generator --region=asia-northeast1
```

View metrics in Cloud Console:
- [Cloud Run Console](https://console.cloud.google.com/run)
- [Cloud Storage Console](https://console.cloud.google.com/storage)

## License

MIT