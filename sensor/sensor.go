package sensor

import (
	"time"
	"github.com/stianeikeland/go-rpio/v4"
)

// Listen on pin 'p' for state-changes, allowing the given amount of
// time for the pin's state to settle.  Returns a channel which will
// send a 'true' every time it transitions (after this settling) to a
// high value, and a 'false' every time it transitions to a low value.
func ListenSensor(p rpio.Pin, settle time.Duration) <-chan bool {

	ch := make(chan bool)

	go func() {
		last_state := false
		state := false
		state_sent := false

		for {
			last_state = state
			state = p.Read() == rpio.High

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
	}()

	return ch
}
