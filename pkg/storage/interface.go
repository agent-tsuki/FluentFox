// Package storage defines the Storage interface for object storage operations.
// No AWS SDK or R2-specific types leak outside this package — callers receive
// plain strings (pre-signed URLs) and errors only.
package storage

import "context"

// Storage is the interface every object storage backend must implement.
type Storage interface {
	// GeneratePresignedUpload returns a pre-signed PUT URL that a client can use
	// to upload a file directly to storage without going through the API server.
	GeneratePresignedUpload(ctx context.Context, key string, contentType string, expiresInSecs int) (string, error)

	// GeneratePresignedDownload returns a pre-signed GET URL for reading a private object.
	GeneratePresignedDownload(ctx context.Context, key string, expiresInSecs int) (string, error)

	// Delete removes an object from the bucket by its key.
	Delete(ctx context.Context, key string) error
}
