package rentals

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseRealRentalsPage(t *testing.T) {
	html, err := os.ReadFile("../../tests/rentals.html")
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(html)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConsumableCapacity != 300 || got.CarriedPoints != 3870 || !got.IndividualPaymentEnabled {
		t.Fatalf("point account = (%d, %d, %t), want (300, 3870, true)", got.ConsumableCapacity, got.CarriedPoints, got.IndividualPaymentEnabled)
	}
	if len(got.Rentals) != 7 {
		t.Fatalf("got %d rentals, want 7", len(got.Rentals))
	}

	first := Rental{
		State:       "USING",
		MonthlyCost: 1090,
		PaidUpDate:  time.Date(2028, time.November, 14, 0, 0, 0, 0, time.UTC),
	}
	if got.Rentals[0] != first {
		t.Errorf("first rental = %#v, want %#v", got.Rentals[0], first)
	}

	returned := Rental{
		State:       "RETURNED",
		MonthlyCost: 600,
		PaidUpDate:  time.Date(2027, time.June, 5, 0, 0, 0, 0, time.UTC),
		PickupDate:  time.Date(2026, time.April, 28, 0, 0, 0, 0, time.UTC),
	}
	var returnedRental Rental
	var usingCost int
	for _, rental := range got.Rentals {
		if rental.State == returned.State {
			returnedRental = rental
		}
		if rental.State == StateUsing {
			usingCost += rental.MonthlyCost
		}
		if rental.State != returned.State && !rental.PickupDate.IsZero() {
			t.Errorf("rental %q without return requests has pickup date %s", rental.State, rental.PickupDate)
		}
	}
	if returnedRental != returned {
		t.Errorf("returned rental = %#v, want %#v", returnedRental, returned)
	}
	if usingCost != 4700 {
		t.Errorf("USING monthly cost sum = %d, want 4700", usingCost)
	}
}

func TestParsePickupDateSelection(t *testing.T) {
	tests := []struct {
		name     string
		requests string
		want     time.Time
	}{
		{
			name:     "ignore canceled request",
			requests: `{"node":{"canceled":true,"pickupDate":"2025-01-01"}}`,
		},
		{
			name: "choose earliest non-canceled request",
			requests: `{"node":{"canceled":false,"pickupDate":"2026-06-20"}},` +
				`{"node":{"canceled":true,"pickupDate":"2026-01-01"}},` +
				`{"node":{"canceled":false,"pickupDate":"2026-04-28"}}`,
			want: time.Date(2026, time.April, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "ignore null pickup date",
			requests: `{"node":{"canceled":false,"pickupDate":null}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(minimalHTML(minimalRentalsRoute("2025-01-01", tt.requests), validAppRoute))
			if err != nil {
				t.Fatal(err)
			}
			if got.Rentals[0].PickupDate != tt.want {
				t.Errorf("PickupDate = %s, want %s", got.Rentals[0].PickupDate, tt.want)
			}
		})
	}
}

func TestParseMissingRentalsEdges(t *testing.T) {
	route := `{"viewer":{"pointAccount":{"consumableCapacity":300},"organization":{"remotehqConfig":{"isIndividualPaymentEnabled":true}}}}`
	got, err := Parse(minimalHTML(route, validAppRoute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rentals) != 0 {
		t.Errorf("got %d rentals, want none", len(got.Rentals))
	}
}

func TestParseErrors(t *testing.T) {
	validRentals := minimalRentalsRoute("2025-01-01", "")
	tests := []struct {
		name string
		html []byte
		want string
	}{
		{
			name: "missing marker",
			html: []byte("<html><body>login</body></html>"),
			want: "remix context marker",
		},
		{
			name: "malformed JSON",
			html: []byte(`<script>window.__remixContext = {not json};</script>`),
			want: "decode Remix context",
		},
		{
			name: "rentals route missing",
			html: minimalHTML("", validAppRoute),
			want: rentalsRouteID,
		},
		{
			name: "app route missing",
			html: minimalHTML(validRentals, ""),
			want: appRouteID,
		},
		{
			name: "consumable capacity missing",
			html: minimalHTML(`{"viewer":{"pointAccount":{},"organization":{"remotehqConfig":{"isIndividualPaymentEnabled":true}}}}`, validAppRoute),
			want: "consumableCapacity",
		},
		{
			name: "individual payment setting missing",
			html: minimalHTML(`{"viewer":{"pointAccount":{"consumableCapacity":300},"organization":{"remotehqConfig":{}}}}`, validAppRoute),
			want: "isIndividualPaymentEnabled",
		},
		{
			name: "carried points missing",
			html: minimalHTML(validRentals, `{"data":{"viewer":{"pointAccount":{}}}}`),
			want: "freeBalanceForOverCapacityRental",
		},
		{
			name: "malformed paid-up date",
			html: minimalHTML(minimalRentalsRoute("2025-02-30", ""), validAppRoute),
			want: "paidUpDate",
		},
		{
			name: "malformed pickup date",
			html: minimalHTML(minimalRentalsRoute("2025-01-31", `{"node":{"canceled":false,"pickupDate":"2025-02-30"}}`), validAppRoute),
			want: "pickupDate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.html)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Parse() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

const validAppRoute = `{"data":{"viewer":{"pointAccount":{"freeBalanceForOverCapacityRental":5}}}}`

func minimalRentalsRoute(paidUpDate, requests string) string {
	return `{"viewer":{"pointAccount":{"consumableCapacity":300},"rentals":{"edges":[{"node":{"state":"USING","monthlyTotalCost":100,"paidUpDate":"` + paidUpDate + `","returnRequests":{"edges":[` + requests + `]}}}]},"organization":{"remotehqConfig":{"isIndividualPaymentEnabled":true}}}}`
}

func minimalHTML(rentalsRoute, appRoute string) []byte {
	var routes []string
	if rentalsRoute != "" {
		routes = append(routes, `"`+rentalsRouteID+`":`+rentalsRoute)
	}
	if appRoute != "" {
		routes = append(routes, `"`+appRouteID+`":`+appRoute)
	}
	return []byte(`<script>window.__remixContext = {"state":{"loaderData":{` + strings.Join(routes, ",") + `}}};</script><p>rest of document</p>`)
}
