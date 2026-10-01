package order

import (
	"testing"

	"github.com/stretchr/testify/assert"

	orderv1 "github.com/geoo115/E-commerceMicroservices/pkg/pb/order/v1"
)

func TestCanTransition(t *testing.T) {
	allowed := map[[2]Status]bool{
		{StatusPending, StatusConfirmed}:   true,
		{StatusPending, StatusCancelled}:   true,
		{StatusConfirmed, StatusPaid}:      true,
		{StatusConfirmed, StatusCancelled}: true,
		{StatusPaid, StatusShipped}:        true,
		{StatusShipped, StatusDelivered}:   true,
	}
	all := []Status{StatusPending, StatusConfirmed, StatusPaid, StatusShipped, StatusDelivered, StatusCancelled}

	for _, from := range all {
		for _, to := range all {
			want := allowed[[2]Status{from, to}]
			assert.Equal(t, want, CanTransition(from, to), "%s -> %s", from, to)
		}
	}
}

func TestStatusProtoRoundTrip(t *testing.T) {
	for s := range toProtoStatus {
		back, ok := StatusFromProto(s.Proto())
		assert.True(t, ok)
		assert.Equal(t, s, back)
	}
	_, ok := StatusFromProto(orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED)
	assert.False(t, ok)
}
