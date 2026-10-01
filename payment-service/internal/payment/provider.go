package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrDeclined is returned by a Provider when the charge is refused.
var ErrDeclined = errors.New("card declined")

// DeclinedCardLastFour makes SimulatedProvider decline the charge, like the
// test card numbers offered by real payment providers.
const DeclinedCardLastFour = "0002"

// Provider charges a payment method. A real implementation would call Stripe,
// Adyen, etc. with an idempotency key.
type Provider interface {
	Charge(ctx context.Context, p Payment) (transactionID string, err error)
}

// SimulatedProvider approves every charge except cards ending in DeclinedCardLastFour.
type SimulatedProvider struct{}

// Charge implements Provider.
func (SimulatedProvider) Charge(_ context.Context, p Payment) (string, error) {
	if p.CardLastFour == DeclinedCardLastFour {
		return "", ErrDeclined
	}
	return "sim_" + uuid.NewString(), nil
}
