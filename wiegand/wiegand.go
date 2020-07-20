package wiegand

// The wiegand package provides a rudimentary driver to read from
// badge readers using the Wiegand protocol.  It requires a Raspberry
// Pi, the WiringPi library, and cgo, as it relies on the C code in
// wiegand_c.go.

import (
	"time"
)

const data_len = 100

type BadgeRead struct {
	RawBits []byte
	Value uint64
	LengthOK bool
	ParityOK bool
}

// ListenBadges returns a channel that will send every badge scanned.
//
// Currently, this channel is never closed and its goroutine never
// ends.
//
// The pins d0_pin and d1_pin should be given as BCM/GPIO pin numbers.
//
// As this relies on an initial C call, it may simply exit the program
// if it fails to initialize - see wiegand_c.go and its initWiegand()
// call.
func ListenBadges(d0_pin int, d1_pin int) (<-chan BadgeRead, error) {

	initWiegand(d0_pin, d1_pin)
	
	ch := make(chan BadgeRead)
	go func(chan<- BadgeRead) {
		var data [data_len]byte
		var even_check byte = 0
		var odd_check byte = 0
		for {
			// Is there data ready?
			n, ok := badgeCheckRaw(&data)
			if !ok {
				// No data was ready. Wait and poll again.
				<-time.After(time.Millisecond * 10)
				continue
			}

			// Make a BadgeRead and copy the bits we need:
			br := BadgeRead{
				RawBits: make([]byte, n),
				Value: 0,
				LengthOK: false,
				ParityOK: false,
			}
			for i,b := range data[:n] {
				br.RawBits[i] = b
			}

			if n != 26 {
				// If number of bits is wrong, not much else
				// can/should be done. Send it and give up.
				ch <- br
				continue
			}
			br.LengthOK = true

			// Check parity for a 26-bit value:
			even_check = 0
			odd_check = 0
			for j := 0; j < 13; j++ {
				even_check = even_check ^ data[j]
				odd_check  = odd_check  ^ data[j + 13]
			}
			br.ParityOK = (even_check == 0) && (odd_check != 0)

			// Whatever the case, try to read the value:
			var val uint64 = 0
			var mask uint64 = 1
			for j := 24; j > 0; j-- {
				if data[j] > 0 {
					val |= mask
				}
				mask <<= 1
			}
			br.Value = val

			// And finally, send it over the channel:
			ch <- br
		}
		
	}(ch)

	return ch, nil
}
