package main

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"time"

	"hive13/rfid/mqtt"
)

func publish(client MQTT.Client) {
	num := 100
	for i := 0; i < num; i++ {
		text := fmt.Sprintf("Message %d", i)
		fmt.Printf("%s\n", text)
		token := client.Publish("hello/world", 0, false, text)
		//token.Wait()
		_ = token
		time.Sleep(time.Second)
	}
}

func main() {
	cfg := mqtt.Config{
		BrokerAddr: "tcp://172.16.3.39:1883",
		Username: "na",
		Password: "na",
		ClientID: "pi",
		TopicSensor: "hive13/sensor",
		TopicBadge: "hive13/sensor",
	}

	client := mqtt.NewClient(cfg)
	fmt.Printf("Got connection\n")
	
	publish(client)
}
