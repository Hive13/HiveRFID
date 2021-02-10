package mqtt

import (
	"log"
	"time"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	// Address for MQTT broker (e.g. "tcp://foobar.com:1883")
	BrokerAddr string
	// Username for MQTT broker (ignored if empty)
	Username string
	// Password for MQTT broker (ignored if empty)
	Password string
	// Client ID for MQTT broker (ignored if empty)
	ClientID string
	// MQTT topic to which we'll publish sensor readings
	TopicSensor string
	// MQTT topic to which we'll publish badge scans
	TopicBadge string
}

func NewClient(c Config) MQTT.Client {

	opts := MQTT.NewClientOptions()
	opts.AddBroker(c.BrokerAddr)
	opts.SetClientID(c.ClientID)
	opts.SetUsername(c.Username)
	opts.SetPassword(c.Password)
	opts.SetDefaultPublishHandler(
		func(client MQTT.Client, msg MQTT.Message) {
			log.Printf("MQTT: recv topic %s: %s", msg.Topic(), msg.Payload())
		})
	opts.SetOnConnectHandler(
		func(client MQTT.Client) {
			log.Printf("MQTT: connected")
		})
	opts.SetConnectionLostHandler(
		func(client MQTT.Client, err error) {
			log.Printf("MQTT: connection lost: %v", err)
		})
	opts.SetReconnectingHandler(
		func(client MQTT.Client, options *MQTT.ClientOptions) {
			log.Printf("MQTT: reconnecting")
		})
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)	
	client := MQTT.NewClient(opts)

	go func(client MQTT.Client) {
		for {
			token := client.Connect()
			if token.Wait() && token.Error() != nil {
				log.Printf("MQTT: unable to connect, %s", token.Error())
				<-time.After(10 * time.Second)
			} else {
				break
			}
		}
	}(client)
	
	return client
}
