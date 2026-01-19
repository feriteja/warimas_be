package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

func GenerateInvoiceNumber() string {
	now := time.Now().UTC()

	datePart := now.Format("20060102-150405")
	millis := now.Nanosecond() / int(time.Millisecond)

	// 4-digit cryptographic random
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		// fallback: time-based entropy
		n = big.NewInt(now.UnixNano() % 10000)
	}

	return fmt.Sprintf(
		"INV-%s-%03d-%04d",
		datePart,
		millis,
		n.Int64(),
	)
}
