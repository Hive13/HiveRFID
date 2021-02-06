package main

import (
	"log"
	"time"

	"hive13/rfid/sensor"
	
	"github.com/stianeikeland/go-rpio/v4"
)


func main() {
	pin_num := 6

	pin := rpio.Pin(pin_num)
	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()
	pin.Input()
	pin.PullUp()

	settle := 300 * time.Millisecond
	for s := range sensor.ListenSensor(pin, settle) {
		log.Printf("%t", s)
	}
}
