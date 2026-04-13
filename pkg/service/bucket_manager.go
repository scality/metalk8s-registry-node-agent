package service

type BucketManager interface {
	// CreateBucket creates a new bucket in the storage. The bucketName MUST be
	// unique relative to the storage.
	CreateBucket(bucketName string) error

	// ListBuckets lists all the buckets in the storage and returns their
	// bucketNames.
	ListBuckets() ([]string, error)

	// DeleteBucket deletes a bucket, and all its content, from the storage
	// based on its bucketName.
	DeleteBucket(bucketName string) error
}
