package main

import (
	"log"
	"time"

	"hive13/rfid/sensor"
	
	"github.com/warthog618/gpiod"
)

func main() {
	pin_num := 6

	chip, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer chip.Close()

	l, err := chip.RequestLine(pin_num, gpiod.AsInput)
	if err != nil {
		panic(err)
	}
	
	settle := 300 * time.Millisecond
	sensor_chan, err := sensor.ListenSensor(l, settle)
	if err != nil {
		panic(err)
	}
	
	for s := range sensor_chan {
		log.Printf("%t", s)
	}
}
