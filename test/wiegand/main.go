package main

import (
	"log"

	"hive13/rfid/wiegand"
)


func main() {
	d0 := 19
	d1 := 26
	log.Printf("D0=%d D1=%d...", d0, d1)
	badges, err := wiegand.ListenBadges(19, 26)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting")

	for b := range badges {
		log.Printf("Main loop: Scanned badge: %+v", b)
	}
}
