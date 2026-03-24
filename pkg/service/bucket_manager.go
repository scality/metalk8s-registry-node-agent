package service

// BucketManager handles CRUD operations on storage buckets.
type BucketManager interface {
	CreateBucket(bucketName string) error
	ListBuckets() ([]string, error)
	DeleteBucket(bucketName string) error
}
