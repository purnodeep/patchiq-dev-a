package workers

import "github.com/riverqueue/river"

// RegisterWorkers returns a Workers bundle with all registered hub job workers.
// binaryFetchWorker may be nil if MinIO is not configured.
func RegisterWorkers(feedWorker *FeedSyncWorker, binaryFetchWorker *BinaryFetchWorker) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, feedWorker)
	if binaryFetchWorker != nil {
		river.AddWorker(workers, binaryFetchWorker)
	}
	return workers
}
