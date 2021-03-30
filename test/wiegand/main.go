package main

import (
	"log"

	"github.com/warthog618/gpiod"

	"hive13/rfid/wiegand"
)


func main() {
	chip, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		log.Fatal(err)
	}
	defer chip.Close()
	
	d0 := 17
	d1 := 18
	log.Printf("D0=%d D1=%d...", d0, d1)
	badges, err := wiegand.ListenBadges(chip, d0, d1)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting")

	for b := range badges {
		log.Printf("Main loop: Scanned badge: %+v", b)
	}
}
