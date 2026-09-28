package rentals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const remixContextMarker = "window.__remixContext ="

const (
	rentalsRouteID = "routes/remote._app.mypage.rentals._index"
	appRouteID     = "routes/remote._app"
)

type remixContext struct {
	State struct {
		LoaderData map[string]json.RawMessage `json:"loaderData"`
	} `json:"state"`
}

type rentalsRoute struct {
	Viewer struct {
		PointAccount struct {
			ConsumableCapacity *int `json:"consumableCapacity"`
		} `json:"pointAccount"`
		Rentals struct {
			Edges []struct {
				Node rentalNode `json:"node"`
			} `json:"edges"`
		} `json:"rentals"`
		Organization struct {
			RemoteHQConfig struct {
				IsIndividualPaymentEnabled *bool `json:"isIndividualPaymentEnabled"`
			} `json:"remotehqConfig"`
		} `json:"organization"`
	} `json:"viewer"`
}

type rentalNode struct {
	State            string `json:"state"`
	MonthlyTotalCost int    `json:"monthlyTotalCost"`
	PaidUpDate       string `json:"paidUpDate"`
	ReturnRequests   struct {
		Edges []struct {
			Node struct {
				Canceled   bool   `json:"canceled"`
				PickupDate string `json:"pickupDate"`
			} `json:"node"`
		} `json:"edges"`
	} `json:"returnRequests"`
}

type appRoute struct {
	Data struct {
		Viewer struct {
			PointAccount struct {
				FreeBalanceForOverCapacityRental *int `json:"freeBalanceForOverCapacityRental"`
			} `json:"pointAccount"`
		} `json:"viewer"`
	} `json:"data"`
}

// Parse extracts the point account and rentals from a RemoteHQ Remix SSR document.
func Parse(html []byte) (*Snapshot, error) {
	marker := bytes.Index(html, []byte(remixContextMarker))
	if marker < 0 {
		return nil, fmt.Errorf("remix context marker %q not found", remixContextMarker)
	}

	var context remixContext
	decoder := json.NewDecoder(bytes.NewReader(html[marker+len(remixContextMarker):]))
	if err := decoder.Decode(&context); err != nil {
		return nil, fmt.Errorf("decode Remix context: %w", err)
	}

	rawRentals, ok := context.State.LoaderData[rentalsRouteID]
	if !ok {
		return nil, fmt.Errorf("loader route %q is missing", rentalsRouteID)
	}
	var route rentalsRoute
	if err := json.Unmarshal(rawRentals, &route); err != nil {
		return nil, fmt.Errorf("decode loader route %q: %w", rentalsRouteID, err)
	}

	if route.Viewer.PointAccount.ConsumableCapacity == nil {
		return nil, fmt.Errorf("loader route %q is missing viewer.pointAccount.consumableCapacity", rentalsRouteID)
	}
	if route.Viewer.Organization.RemoteHQConfig.IsIndividualPaymentEnabled == nil {
		return nil, fmt.Errorf("loader route %q is missing viewer.organization.remotehqConfig.isIndividualPaymentEnabled", rentalsRouteID)
	}

	rawApp, ok := context.State.LoaderData[appRouteID]
	if !ok {
		return nil, fmt.Errorf("loader route %q is missing", appRouteID)
	}
	var app appRoute
	if err := json.Unmarshal(rawApp, &app); err != nil {
		return nil, fmt.Errorf("decode loader route %q: %w", appRouteID, err)
	}
	if app.Data.Viewer.PointAccount.FreeBalanceForOverCapacityRental == nil {
		return nil, fmt.Errorf("loader route %q is missing data.viewer.pointAccount.freeBalanceForOverCapacityRental", appRouteID)
	}

	snapshot := &Snapshot{
		ConsumableCapacity:       *route.Viewer.PointAccount.ConsumableCapacity,
		CarriedPoints:            *app.Data.Viewer.PointAccount.FreeBalanceForOverCapacityRental,
		IndividualPaymentEnabled: *route.Viewer.Organization.RemoteHQConfig.IsIndividualPaymentEnabled,
		Rentals:                  make([]Rental, 0, len(route.Viewer.Rentals.Edges)),
	}
	for i, edge := range route.Viewer.Rentals.Edges {
		node := edge.Node
		paidUpDate, err := time.Parse(time.DateOnly, node.PaidUpDate)
		if err != nil {
			return nil, fmt.Errorf("rental %d paidUpDate %q: %w", i, node.PaidUpDate, err)
		}

		rental := Rental{
			State:       node.State,
			MonthlyCost: node.MonthlyTotalCost,
			PaidUpDate:  paidUpDate,
		}
		for j, requestEdge := range node.ReturnRequests.Edges {
			request := requestEdge.Node
			if request.Canceled || request.PickupDate == "" {
				continue
			}
			pickupDate, err := time.Parse(time.DateOnly, request.PickupDate)
			if err != nil {
				return nil, fmt.Errorf("rental %d return request %d pickupDate %q: %w", i, j, request.PickupDate, err)
			}
			if rental.PickupDate.IsZero() || pickupDate.Before(rental.PickupDate) {
				rental.PickupDate = pickupDate
			}
		}
		snapshot.Rentals = append(snapshot.Rentals, rental)
	}

	return snapshot, nil
}
