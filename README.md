# jplaw2epub-generate-epub-function

Cloud Function for generating EPUB files from Japanese law data asynchronously.

## Overview

This Cloud Function processes EPUB generation requests from the main [jplaw2epub-web-api](https://github.com/ngs/jplaw2epub-web-api) service. It handles the heavy EPUB generation process separately to avoid timeout issues.

## Architecture

```
Main API → Cloud Function → Cloud Storage
             (this repo)      (EPUB files)
```

## Setup

### Prerequisites

- Google Cloud Project with billing enabled
- Cloud Functions API enabled
- Cloud Storage API enabled
- Service account with appropriate permissions

### Environment Variables

- `EPUB_BUCKET_NAME`: Cloud Storage bucket name (default: `epub-storage`)
- `PROJECT_ID`: GCP Project ID

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Run locally:
```bash
export EPUB_BUCKET_NAME=epub-storage
go run main.go
```

### Deployment

#### Option 1: GitHub Actions (Recommended)

1. Set up GitHub Secrets:
   - `PROJECT_ID`: Your GCP project ID
   - `WIF_PROVIDER`: Workload Identity Federation provider
   - `WIF_SERVICE_ACCOUNT`: Service account for deployment
   - `EPUB_BUCKET_NAME`: Storage bucket name (optional)

2. Push to main branch or trigger manually from Actions tab

#### Option 2: Manual Deployment

```bash
gcloud functions deploy generate-epub \
  --gen2 \
  --runtime=go122 \
  --region=asia-northeast1 \
  --source=. \
  --entry-point=GenerateEpub \
  --trigger-http \
  --allow-unauthenticated \
  --timeout=540s \
  --memory=1GB \
  --max-instances=5 \
  --set-env-vars="EPUB_BUCKET_NAME=epub-storage" \
  --project=YOUR_PROJECT_ID
```

## Storage Setup

Create and configure the storage bucket:

```bash
# Create bucket
gsutil mb -l asia-northeast1 gs://epub-storage

# Set lifecycle (auto-delete after 30 days)
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [{
      "action": {"type": "Delete"},
      "condition": {"age": 30}
    }]
  }
}
EOF
gsutil lifecycle set lifecycle.json gs://epub-storage

# Set CORS
cat > cors.json <<EOF
[{
  "origin": ["*"],
  "method": ["GET", "HEAD"],
  "responseHeader": ["Content-Type"],
  "maxAgeSeconds": 3600
}]
EOF
gsutil cors set cors.json gs://epub-storage
```

## API

### Request

```json
POST /
{
  "id": "335M50000002060_20250601_507M60000002046",
  "version": "v1.0.0"
}
```

### Response

Success:
```json
{
  "status": "success",
  "id": "335M50000002060_20250601_507M60000002046"
}
```

Error:
```
HTTP 500 Internal Server Error
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

- Timeout: 9 minutes (540 seconds)
- Memory: 1GB
- Max instances: 5
- Expected processing time: 30-60 seconds per EPUB

## Cost Estimation

For 1000 EPUB generations per month:
- Cloud Functions: ~$0.50
- Cloud Storage: ~$0.07
- Total: ~$0.57/month

## Monitoring

View logs:
```bash
gcloud functions logs read generate-epub --limit=50
```

View metrics in Cloud Console:
- [Cloud Functions Console](https://console.cloud.google.com/functions)
- [Cloud Storage Console](https://console.cloud.google.com/storage)

## License

MIT