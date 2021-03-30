package main

import (
	"log"
	"time"

	"hive13/rfid/sensor"
	
	"github.com/warthog618/gpiod"
)

func main() {
	pin_num := 6

	c, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	settle := 300 * time.Millisecond
	sensor, err := sensor.ListenSensor(c, pin_num, settle)
	if err != nil {
		panic(err)
	}
	
	for s := range sensor {
		log.Printf("%t", s)
	}
}
