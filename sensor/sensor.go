package sensor

import (
	"time"
	"log"
	"github.com/warthog618/gpiod"
)

// Listen on pin 'p' for state-changes, allowing the given amount of
// time for the pin's state to settle.  Returns a channel which will
// send a 'true' every time it transitions (after this settling) to a
// high value, and a 'false' every time it transitions to a low value.
func ListenSensor(pin *gpiod.Line, settle time.Duration) (<-chan bool, error) {

	ch := make(chan bool)

	go func(pin *gpiod.Line) {
		last_state := false
		state := false
		state_sent := false
		val := 0
		var err error
		
		for {
			last_state = state
			val, err = pin.Value()
			if err != nil {
				log.Printf("Error reading GPIO pin %d for sensor: %s", err)
			} else {
				state = val == 1

				if state != last_state {
					<-time.After(settle)
				} else {
					if state != state_sent {
						ch <- state
						state_sent = state
					}
					<-time.After(10 * time.Millisecond)
				}
			}
		}
	}(pin)

	return ch, nil
}
