package simulate

import (
	"errors"
	"time"

	"remotehq-simulator/internal/rentals"
)

// Month is the point usage and payment for one month of the new rental.
type Month struct {
	Number            int
	Date              time.Time
	Available         int
	FromCapacity      int
	FromCarried       int
	IndividualPayment int
	CarriedBalance    int
}

// Result summarizes point usage and payments over the rental term.
type Result struct {
	Months                 []Month
	PointsOnlyLimit        int
	TotalFromCapacity      int
	TotalFromCarried       int
	TotalIndividualPayment int
}

// Run simulates a new rental using the point balance and existing rentals in s.
func Run(s *rentals.Snapshot, monthlyPoints, months int, start time.Time) (Result, error) {
	if monthlyPoints <= 0 {
		return Result{}, errors.New("monthly points must be positive")
	}
	if months <= 0 {
		return Result{}, errors.New("months must be positive")
	}

	result := Result{Months: make([]Month, 0, months)}
	carried := s.CarriedPoints
	for k := range months {
		date := monthDate(start, k)
		available := s.ConsumableCapacity
		for _, rental := range s.Rentals {
			if rental.State == rentals.StateUsing && !rental.OccupiesAt(date) {
				available += rental.MonthlyCost
			}
		}

		month := Month{
			Number:       k + 1,
			Date:         date,
			Available:    available,
			FromCapacity: min(monthlyPoints, available),
		}
		if monthlyPoints <= available {
			carried += available - monthlyPoints
		} else {
			shortfall := monthlyPoints - available
			month.FromCarried = min(carried, shortfall)
			carried -= month.FromCarried
			month.IndividualPayment = shortfall - month.FromCarried
		}
		month.CarriedBalance = carried
		result.Months = append(result.Months, month)
		result.TotalFromCapacity += month.FromCapacity
		result.TotalFromCarried += month.FromCarried
		result.TotalIndividualPayment += month.IndividualPayment
	}

	result.PointsOnlyLimit = result.Months[0].Available + s.CarriedPoints/months
	return result, nil
}

func monthDate(start time.Time, offset int) time.Time {
	first := time.Date(start.Year(), start.Month()+time.Month(offset), 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := min(start.Day(), lastDay)
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
}
