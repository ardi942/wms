package stockrequest

func DeriveStatus(requested, issued int) string {
	if issued <= 0 {
		return StatusPending
	}
	if issued < requested {
		return StatusPartiallyFulfilled
	}
	return StatusFulfilled
}

func IsTerminal(status string) bool {
	return status == StatusFulfilled || status == StatusRejected || status == StatusCanceled
}
