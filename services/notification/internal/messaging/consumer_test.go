package messaging

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCountReadsTheHeaderWhateverItsIntegerType(t *testing.T) {
	tests := map[string]struct {
		headers amqp.Table
		want    int
	}{
		"missing": {nil, 0},
		"int32":   {amqp.Table{retryCountHeader: int32(2)}, 2},
		"int64":   {amqp.Table{retryCountHeader: int64(3)}, 3},
		"garbage": {amqp.Table{retryCountHeader: "x"}, 0},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := retryCountOf(amqp.Delivery{Headers: test.headers}); got != test.want {
				t.Errorf("retry count = %d, want %d", got, test.want)
			}
		})
	}
}
