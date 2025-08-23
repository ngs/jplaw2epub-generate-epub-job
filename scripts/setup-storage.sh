#!/bin/bash

# Cloud Storage setup script for EPUB storage

PROJECT_ID=${PROJECT_ID:-$(gcloud config get-value project)}
BUCKET_NAME=${EPUB_BUCKET_NAME:-${PROJECT_ID}-epub-storage}
REGION=${REGION:-asia-northeast1}

echo "Setting up Cloud Storage for EPUB storage"
echo "Project: $PROJECT_ID"
echo "Bucket: $BUCKET_NAME"
echo "Region: $REGION"

# Create the bucket if it doesn't exist
if ! gcloud storage buckets describe gs://$BUCKET_NAME &>/dev/null; then
  echo "Creating bucket gs://$BUCKET_NAME..."
  gcloud storage buckets create gs://$BUCKET_NAME \
    --project=$PROJECT_ID \
    --location=$REGION \
    --uniform-bucket-level-access
else
  echo "Bucket gs://$BUCKET_NAME already exists"
fi

# Skip lifecycle configuration (no automatic deletion)
# cat > /tmp/lifecycle.json <<EOF
# {
#   "lifecycle": {
#     "rule": []
#   }
# }
# EOF
# 
# echo "Setting lifecycle rules..."
# gcloud storage buckets update gs://$BUCKET_NAME --lifecycle-file=/tmp/lifecycle.json

# Set CORS configuration for signed URLs
cat > /tmp/cors.json <<EOF
[
  {
    "origin": ["*"],
    "method": ["GET", "HEAD"],
    "responseHeader": ["Content-Type"],
    "maxAgeSeconds": 3600
  }
]
EOF

echo "Setting CORS configuration..."
gcloud storage buckets update gs://$BUCKET_NAME --cors-file=/tmp/cors.json

# Clean up temp files
rm -f /tmp/cors.json

echo "Cloud Storage setup complete!"
echo ""
echo "Bucket name: $BUCKET_NAME"
echo ""
echo "The Cloud Run Job will be automatically deployed via GitHub Actions"
echo "when you push to the master branch."