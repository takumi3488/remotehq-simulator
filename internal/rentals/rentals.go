// Package rentals extracts point and rental state from the RemoteHQ
// "利用状況・返却管理" page (https://app.hq-hq.co.jp/remote/mypage/rentals).
package rentals

import "time"

// StateUsing is the rental state that consumes monthly granted points.
const StateUsing = "USING"

// Snapshot is the point/rental state captured from the rentals page.
type Snapshot struct {
	// ConsumableCapacity is 残レンタル可能ポイント (pt/月): granted monthly
	// points not occupied by current rentals.
	ConsumableCapacity int
	// CarriedPoints is 貯まったポイント (未利用ポイント, pt).
	CarriedPoints int
	// IndividualPaymentEnabled reports whether 自己負担 is enabled for the organization.
	IndividualPaymentEnabled bool
	Rentals                  []Rental
}

// Rental is one rental listed on the page. Dates are civil dates at UTC midnight.
type Rental struct {
	State       string
	MonthlyCost int // pt/月
	// PaidUpDate is the last day (inclusive) on which the rental consumes points.
	PaidUpDate time.Time
	// PickupDate is the earliest non-canceled return pickup date; zero if none.
	PickupDate time.Time
}

// OccupiesAt reports whether the rental consumes monthly capacity on date d.
// Capacity is released after PaidUpDate and from PickupDate (返却集荷日) onward.
func (r Rental) OccupiesAt(d time.Time) bool {
	if r.State != StateUsing || d.After(r.PaidUpDate) {
		return false
	}
	return r.PickupDate.IsZero() || d.Before(r.PickupDate)
}
