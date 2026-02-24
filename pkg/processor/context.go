package processor

import "context"

// checkContext performs a non-blocking check on the context's Done channel.
// Returns the context's error if the context has been canceled or timed out,
// otherwise returns nil.
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
