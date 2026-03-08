package model

// ComputeFailureState computes next retry/failure state with shared rules.
func ComputeFailureState(
	currentTimes int,
	success bool,
	threshold int,
	successStatus string,
	failureStatus string,
	thresholdStatus string,
) (times int, status string, reachedThreshold bool) {
	if threshold <= 0 {
		threshold = 1
	}

	if success {
		return 0, successStatus, false
	}

	times = currentTimes + 1
	status = failureStatus
	reachedThreshold = times >= threshold
	if reachedThreshold && thresholdStatus != "" {
		status = thresholdStatus
	}

	return times, status, reachedThreshold
}
