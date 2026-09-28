package simulate

import (
	"reflect"
	"testing"
	"time"

	"remotehq-simulator/internal/rentals"
)

func TestRunIndividualPaymentExample(t *testing.T) {
	result, err := Run(&rentals.Snapshot{CarriedPoints: 300}, 200, 12, date(2026, time.January, 1))
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Months[0]; got.FromCarried != 200 || got.IndividualPayment != 0 || got.CarriedBalance != 100 {
		t.Errorf("month 1 = %+v", got)
	}
	if got := result.Months[1]; got.FromCarried != 100 || got.IndividualPayment != 100 || got.CarriedBalance != 0 {
		t.Errorf("month 2 = %+v", got)
	}
	for i, month := range result.Months[2:] {
		if month.IndividualPayment != 200 {
			t.Errorf("month %d payment = %d, want 200", i+3, month.IndividualPayment)
		}
	}
	if result.TotalIndividualPayment != 2100 || result.TotalFromCarried != 300 {
		t.Errorf("totals = payment %d, carried %d; want 2100, 300", result.TotalIndividualPayment, result.TotalFromCarried)
	}
}

func TestRunCarriedPointsExample(t *testing.T) {
	start := date(2026, time.January, 1)
	tests := []struct {
		name             string
		carried          int
		wantLimit        int
		wantFinalPayment int
	}{
		{name: "1200 points", carried: 1200, wantLimit: 600},
		{name: "1199 points", carried: 1199, wantLimit: 599, wantFinalPayment: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(&rentals.Snapshot{ConsumableCapacity: 500, CarriedPoints: tt.carried}, 600, 12, start)
			if err != nil {
				t.Fatal(err)
			}
			if result.PointsOnlyLimit != tt.wantLimit {
				t.Errorf("PointsOnlyLimit = %d, want %d", result.PointsOnlyLimit, tt.wantLimit)
			}
			for i, month := range result.Months {
				if month.FromCapacity != 500 || month.FromCarried != 100 || month.IndividualPayment != 0 {
					if tt.carried == 1199 && i == 11 {
						if month.FromCapacity != 500 || month.FromCarried != 99 || month.IndividualPayment != tt.wantFinalPayment {
							t.Errorf("month 12 = %+v", month)
						}
						continue
					}
					t.Errorf("month %d = %+v", i+1, month)
				}
			}
			if result.Months[11].CarriedBalance != 0 {
				t.Errorf("final carried balance = %d, want 0", result.Months[11].CarriedBalance)
			}
		})
	}
}

func TestRunPointsOnlyLimits(t *testing.T) {
	for _, tt := range []struct {
		months int
		want   int
	}{{12, 622}, {24, 461}, {36, 407}} {
		t.Run(time.Duration(tt.months).String(), func(t *testing.T) {
			result, err := Run(&rentals.Snapshot{ConsumableCapacity: 300, CarriedPoints: 3870}, 1, tt.months, date(2026, time.January, 1))
			if err != nil {
				t.Fatal(err)
			}
			if result.PointsOnlyLimit != tt.want {
				t.Errorf("PointsOnlyLimit = %d, want %d", result.PointsOnlyLimit, tt.want)
			}
		})
	}
}

func TestRunAccruesSurplus(t *testing.T) {
	result, err := Run(&rentals.Snapshot{ConsumableCapacity: 500}, 300, 3, date(2026, time.January, 1))
	if err != nil {
		t.Fatal(err)
	}
	for i, month := range result.Months {
		if month.CarriedBalance != (i+1)*200 {
			t.Errorf("month %d balance = %d, want %d", i+1, month.CarriedBalance, (i+1)*200)
		}
	}
}

func TestRunReleasesCapacity(t *testing.T) {
	start := date(2026, time.January, 1)
	february := date(2026, time.February, 1)
	february15 := date(2026, time.February, 15)
	march := date(2026, time.March, 1)
	tests := []struct {
		name      string
		rental    rentals.Rental
		available []int
		payments  []int
	}{
		{
			name:      "paid up after February",
			rental:    rentals.Rental{State: rentals.StateUsing, MonthlyCost: 300, PaidUpDate: february15},
			available: []int{0, 0, 300},
			payments:  []int{400, 400, 100},
		},
		{
			name:      "pickup on February first",
			rental:    rentals.Rental{State: rentals.StateUsing, MonthlyCost: 300, PaidUpDate: march, PickupDate: february},
			available: []int{0, 300, 300},
			payments:  []int{400, 100, 100},
		},
		{
			name:      "non-using states do not add capacity",
			rental:    rentals.Rental{State: "RETURNED", MonthlyCost: 300, PaidUpDate: start},
			available: []int{0, 0, 0},
			payments:  []int{400, 400, 400},
		},
		{
			name:      "paid up state does not add capacity",
			rental:    rentals.Rental{State: "PAID_UP", MonthlyCost: 300, PaidUpDate: start},
			available: []int{0, 0, 0},
			payments:  []int{400, 400, 400},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(&rentals.Snapshot{Rentals: []rentals.Rental{tt.rental}}, 400, 3, start)
			if err != nil {
				t.Fatal(err)
			}
			for i, month := range result.Months {
				if month.Available != tt.available[i] || month.IndividualPayment != tt.payments[i] {
					t.Errorf("month %d available/payment = %d/%d, want %d/%d", i+1, month.Available, month.IndividualPayment, tt.available[i], tt.payments[i])
				}
			}
		})
	}
}

func TestRunClampsMonthEndFromStart(t *testing.T) {
	result, err := Run(&rentals.Snapshot{}, 1, 3, date(2026, time.January, 31))
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		date(2026, time.January, 31),
		date(2026, time.February, 28),
		date(2026, time.March, 31),
	}
	for i, month := range result.Months {
		if !month.Date.Equal(want[i]) {
			t.Errorf("month %d date = %s, want %s", i+1, month.Date.Format(time.DateOnly), want[i].Format(time.DateOnly))
		}
	}
}

func TestRunRejectsInvalidInputs(t *testing.T) {
	for _, tt := range []struct {
		name          string
		monthlyPoints int
		months        int
	}{
		{name: "zero monthly points", monthlyPoints: 0, months: 1},
		{name: "negative monthly points", monthlyPoints: -1, months: 1},
		{name: "zero months", monthlyPoints: 1, months: 0},
		{name: "negative months", monthlyPoints: 1, months: -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Run(&rentals.Snapshot{}, tt.monthlyPoints, tt.months, date(2026, time.January, 1)); err == nil {
				t.Fatal("Run returned no error")
			}
		})
	}
}

func TestRunDoesNotMutateSnapshot(t *testing.T) {
	snapshot := rentals.Snapshot{
		ConsumableCapacity: 500,
		CarriedPoints:      100,
		Rentals: []rentals.Rental{{
			State: rentals.StateUsing, MonthlyCost: 300, PaidUpDate: date(2026, time.January, 31),
		}},
	}
	before := rentals.Snapshot{
		ConsumableCapacity:       snapshot.ConsumableCapacity,
		CarriedPoints:            snapshot.CarriedPoints,
		IndividualPaymentEnabled: snapshot.IndividualPaymentEnabled,
		Rentals:                  append([]rentals.Rental(nil), snapshot.Rentals...),
	}
	if _, err := Run(&snapshot, 600, 2, date(2026, time.January, 1)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, before) {
		t.Errorf("snapshot mutated: got %+v, want %+v", snapshot, before)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
