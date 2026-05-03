package domain

const (
	MaxRetries      = 5
	MinPriority     = 0
	MaxPriority     = 100
	DefaultPriority = 50
)

// ValidatePriority checks if priority is within valid range (0-100)
func ValidatePriority(priority int) error {
	if priority < MinPriority || priority > MaxPriority {
		return ErrInvalidInput
	}
	return nil
}

// CanRetry checks if a job can be retried (max 5 retries)
func (j *GenerationJob) CanRetry() bool {
	if j == nil {
		return false
	}
	return j.RetryCount < MaxRetries
}

// GetRetryCount returns how many times this job has been retried
func (j *GenerationJob) GetRetryCount() int {
	if j == nil {
		return 0
	}
	return j.RetryCount
}

// IsRetryJob returns true if this job is a retry of another job
func (j *GenerationJob) IsRetryJob() bool {
	if j == nil {
		return false
	}
	return j.ParentJobID != nil && *j.ParentJobID != ""
}
