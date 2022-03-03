package accounting

import (
	"time"

	"github.com/penguintop/penguin/pkg/penguin"
)

func (s *Accounting) SetTimeNow(f func() time.Time) {
	s.timeNow = f
}

func (s *Accounting) SetTime(k int64) {
	s.SetTimeNow(func() time.Time {
		return time.Unix(k, 0)
	})
}

func (a *Accounting) IsPaymentOngoing(peer penguin.Address) bool {
	return a.getAccountingPeer(peer).paymentOngoing
}
