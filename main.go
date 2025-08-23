package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	jplaw "go.ngs.io/jplaw-api-v2"
	"go.ngs.io/jplaw2epub"
)

func main() {
	var (
		revisionID = flag.String("revision-id", "", "Law revision ID to convert")
		version    = flag.String("version", "v1.0.0", "App version for storage path")
		bucketName = flag.String("bucket", "", "GCS bucket name (defaults to EPUB_BUCKET_NAME env var)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	if *revisionID == "" {
		log.Fatal("Error: --revision-id is required")
	}

	if *bucketName == "" {
		*bucketName = os.Getenv("EPUB_BUCKET_NAME")
		if *bucketName == "" {
			*bucketName = "epub-storage"
		}
	}

	if *verbose {
		log.Printf("Starting EPUB generation for revision ID: %s", *revisionID)
		log.Printf("Version: %s", *version)
		log.Printf("Bucket: %s", *bucketName)
	}

	ctx := context.Background()

	// Update status to PROCESSING.
	statusPath := fmt.Sprintf("%s/%s.status", *version, *revisionID)
	if err := updateStatus(ctx, *bucketName, statusPath, "PROCESSING", ""); err != nil {
		log.Printf("Warning: Failed to update status to PROCESSING: %v", err)
	}

	// Generate EPUB.
	epubData, err := generateEPUBFromID(ctx, *revisionID)
	if err != nil {
		log.Printf("Error: Failed to generate EPUB for %s: %v", *revisionID, err)
		// Update status to FAILED.
		if updateErr := updateStatus(ctx, *bucketName, statusPath, "FAILED", err.Error()); updateErr != nil {
			log.Printf("Warning: Failed to update status to FAILED: %v", updateErr)
		}
		os.Exit(1)
	}

	// Save EPUB to Cloud Storage.
	epubPath := fmt.Sprintf("%s/%s.epub", *version, *revisionID)
	if err := uploadEPUB(ctx, *bucketName, epubPath, epubData); err != nil {
		log.Printf("Error: Failed to upload EPUB: %v", err)
		if updateErr := updateStatus(ctx, *bucketName, statusPath, "FAILED", err.Error()); updateErr != nil {
			log.Printf("Warning: Failed to update status to FAILED: %v", updateErr)
		}
		os.Exit(1)
	}

	// Delete status file (no longer needed).
	if err := deleteObject(ctx, *bucketName, statusPath); err != nil {
		log.Printf("Warning: Failed to delete status file: %v", err)
		// This is not critical, continue.
	}

	log.Printf("Successfully generated EPUB for %s at %s", *revisionID, epubPath)
}

func updateStatus(ctx context.Context, bucketName, path, status, errorMsg string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	obj := client.Bucket(bucketName).Object(path)

	statusData := map[string]string{
		"status":    status,
		"updatedAt": time.Now().Format(time.RFC3339),
	}
	if errorMsg != "" {
		statusData["error"] = errorMsg
	}

	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	if err := json.NewEncoder(w).Encode(statusData); err != nil {
		return err
	}
	return w.Close()
}

func uploadEPUB(ctx context.Context, bucketName, path string, data []byte) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	obj := client.Bucket(bucketName).Object(path)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/epub+zip"

	if _, err := w.Write(data); err != nil {
		return err
	}
	return w.Close()
}

func deleteObject(ctx context.Context, bucketName, path string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	obj := client.Bucket(bucketName).Object(path)
	return obj.Delete(ctx)
}

func generateEPUBFromID(ctx context.Context, lawIdOrNumOrRevisionId string) ([]byte, error) {
	log.Printf("Fetching law data for ID: %s", lawIdOrNumOrRevisionId)

	client := jplaw.NewClient()

	xmlFormat := jplaw.ResponseFormatXml
	params := &jplaw.GetLawDataParams{
		LawFullTextFormat: &xmlFormat,
	}

	lawData, err := client.GetLawData(lawIdOrNumOrRevisionId, params)
	if err != nil {
		return nil, fmt.Errorf("error fetching law data: %v", err)
	}

	xmlContent, err := extractXMLContent(lawData, lawIdOrNumOrRevisionId)
	if err != nil {
		return nil, fmt.Errorf("error extracting XML content: %v", err)
	}

	xmlReader := bytes.NewReader(xmlContent)
	options := &jplaw2epub.EPUBOptions{}

	// Check if this is a revision ID (has 3 components separated by _).
	idComponents := strings.Split(lawIdOrNumOrRevisionId, "_")
	if len(idComponents) == 3 {
		options.RevisionID = lawIdOrNumOrRevisionId
		options.APIClient = client
	}

	book, err := jplaw2epub.CreateEPUBFromXMLFileWithOptions(xmlReader, options)
	if err != nil {
		return nil, fmt.Errorf("error creating EPUB: %v", err)
	}

	var buf bytes.Buffer
	if _, err := book.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("error writing EPUB to buffer: %v", err)
	}

	log.Printf("Successfully converted law ID %s to EPUB (%d bytes)", lawIdOrNumOrRevisionId, buf.Len())
	return buf.Bytes(), nil
}

func extractXMLContent(lawData *jplaw.LawDataResponse, lawID string) ([]byte, error) {
	if lawData.LawFullText == nil {
		return nil, fmt.Errorf("no law content in response")
	}

	xmlStr, ok := (*lawData.LawFullText).(string)
	if !ok {
		return nil, fmt.Errorf("invalid XML format in response")
	}

	decodedXML, err := base64.StdEncoding.DecodeString(xmlStr)
	if err != nil {
		return nil, fmt.Errorf("error decoding XML content: %v", err)
	}

	xmlContent := string(decodedXML)
	if strings.HasPrefix(xmlContent, "<TmpRootTag>") {
		xmlContent = strings.TrimPrefix(xmlContent, "<TmpRootTag>")
		xmlContent = strings.TrimSuffix(xmlContent, "</TmpRootTag>")
	}

	log.Printf("Decoded XML content length for law ID %s: %d bytes", lawID, len(xmlContent))
	return []byte(xmlContent), nil
}