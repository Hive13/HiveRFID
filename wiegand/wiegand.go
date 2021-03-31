package wiegand

// The wiegand package provides a rudimentary driver to read from
// badge readers using the Wiegand protocol.  It requires a Raspberry
// Pi, the WiringPi library, and cgo, as it relies on the C code in
// wiegand_c.go.

import (
	"time"

	"github.com/warthog618/gpiod"
)

const data_len = 100
const max_wiegand_bits = 32
const reader_timeout = 3000000

var _isr_data [max_wiegand_bits]bool
var _isr_bit_count uint64
var _isr_bit_time time.Time

type BadgeRead struct {
	RawBits []byte
	Value uint64
	LengthOK bool
	ParityOK bool
}

func d0_fall_isr(evt gpiod.LineEvent) {
	if evt.Type == gpiod.LineEventFallingEdge {
		if _isr_bit_count < max_wiegand_bits {
			_isr_bit_count += 1
		}
		_isr_bit_time = time.Now()
	}
}

func d1_fall_isr(evt gpiod.LineEvent) {
	if evt.Type == gpiod.LineEventFallingEdge {
		if _isr_bit_count < max_wiegand_bits {
			_isr_data[_isr_bit_count] = true
			_isr_bit_count += 1
		}
		_isr_bit_time = time.Now()
	}
}

func reset() {
	for i,_ := range _isr_data {
		_isr_data[i] = false
	}
	_isr_bit_count = 0
}

func pending_bit_count() uint64 {
	delta := time.Now().Sub(_isr_bit_time)
	if delta.Nanoseconds() > reader_timeout {
		return _isr_bit_count
	}
	return 0
}

func badgeCheckRaw(data *[data_len]byte) (uint64, bool) {
	if pending_bit_count() == 0 {
		return 0, false
	}

	count := _isr_bit_count
	if count > data_len {
		count = data_len
	}
	for i := uint64(0); i < count; i += 1 {
		if _isr_data[i] {
			data[i] = 1
		} else {
			data[i] = 0
		}
	}
	reset()
	return count, true
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
func ListenBadges(chip *gpiod.Chip, d0_pin int, d1_pin int) (<-chan BadgeRead, error) {

	d0, err := chip.RequestLine(d0_pin,
		gpiod.WithFallingEdge,
		gpiod.WithEventHandler(d0_fall_isr))
	if err != nil {
		return nil, err
	}

	d1, err := chip.RequestLine(d1_pin,
		gpiod.WithFallingEdge,
		gpiod.WithEventHandler(d1_fall_isr))
	if err != nil {
		d0.Close()
		return nil, err
	}
	
	ch := make(chan BadgeRead)
	go func(chan<- BadgeRead) {
		defer d0.Close()
		defer d1.Close()
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
