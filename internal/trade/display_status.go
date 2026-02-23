package trade

// DisplayStatus is the API-facing status derived from TradeStatus and time.
// It is computed by DisplayStatusPolicy, not stored in the database.
type DisplayStatus string

const (
	DisplayStatusLocking         DisplayStatus = "locking"
	DisplayStatusReadyToSettle   DisplayStatus = "ready_to_settle"
	DisplayStatusExpiredRefundable DisplayStatus = "expired_refundable"
	DisplayStatusSettled         DisplayStatus = "settled"
	DisplayStatusRefunded        DisplayStatus = "refunded"
)

// DisplayStatusPolicy computes DisplayStatus from a Trade and current time.
// grace is the duration after endTime before refund becomes available.
type DisplayStatusPolicy struct {
	GraceSeconds int64
}

// Compute returns the DisplayStatus for a trade at the given timestamp (unix seconds).
func (p *DisplayStatusPolicy) Compute(t *Trade, nowUnix int64) DisplayStatus {
	switch t.Status {
	case TradeStatusSettled:
		return DisplayStatusSettled
	case TradeStatusRefunded:
		return DisplayStatusRefunded
	case TradeStatusOpen:
		if nowUnix < t.EndTime {
			return DisplayStatusLocking
		}
		refundAvailableAt := t.EndTime + p.GraceSeconds
		if nowUnix < refundAvailableAt {
			return DisplayStatusReadyToSettle
		}
		return DisplayStatusExpiredRefundable
	default:
		return DisplayStatusLocking
	}
}
