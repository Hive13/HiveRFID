package main

import (
	"log"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)


func main() {
	pin_num := 23

	pin := rpio.Pin(pin_num)
	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()
	pin.Input()
	pin.PullUp()

	settle := 300 * time.Millisecond

	for s := range ListenSensor(pin, settle) {
		log.Printf("%t", s)
	}
}

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
