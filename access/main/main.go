package main

// Commandline for RFID/door access server. This turns arguments to a
// configuration, but is not responsible for any of the actual logic.

import (
	"log"
	"time"
	
	"github.com/spf13/cobra"

	"hive13/rfid/access"
)

var cfg *access.Config
var device_key string
var hold_msec int
var cache_hours int

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}


// Cobra boilerplate:
var rootCmd = &cobra.Command{
	Use:   "access",
	Short: "Start Intweb RFID & door access server",
	Run: func(_ *cobra.Command, args []string) {

		if cfg == nil {
			log.Panic("Configuration never initialized")
		}

		// Convert this data to the right format (everything else is
		// fine but Cobra can't read bytestrings or durations directly):
		cfg.IntwebDeviceKey = []byte(device_key)
		cfg.LockHoldTime = time.Duration(hold_msec) * time.Millisecond
		cfg.BadgeCacheTime = time.Duration(cache_hours) * time.Hour
		
		log.Printf("%+v", cfg)

		// We have a configuration. Go run the server.
		access.Run(cfg)
	},
}

func init() {
	cfg = &access.Config{
	}

	rootCmd.PersistentFlags().StringVar(&cfg.GpioDev, "gpio", "gpiochip0",
		"GPIO character device, e.g. gpiochip0 for /dev/gpiochip0")
	rootCmd.PersistentFlags().IntVar(&cfg.PinD0, "d0", 17,
		"BCM/GPIO input pin number for badge reader's Wiegand D0 pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinD1, "d1", 18,
		"BCM/GPIO input pin number for badge reader's Wiegand D1 pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinBeeper, "beeper", 26,
		"BCM/GPIO output pin number for badge reader's beeper pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinLED, "led", 16,
		"BCM/GPIO output pin number for badge reader's LED pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinSensor, "sensor", -1,
		"BCM/GPIO input pin number for door sensor; if -1, disable")
	rootCmd.PersistentFlags().BoolVar(&cfg.SensorPolarity, "polarity", true,
		"If true, open door = sensor high, closed = low. If false, open = low, closed = high.")

	rootCmd.PersistentFlags().IntVar(&cfg.PinLock, "lock", 24,
		"BCM/GPIO output pin number to control door lock/latch")

	rootCmd.PersistentFlags().IntVar(&hold_msec, "hold", 3000,
		"Time in milliseconds for which to hold lock open")

	rootCmd.PersistentFlags().IntVar(&cache_hours, "cache-time", 96,
		"Time in hours to keep a badge in cache")
	
	rootCmd.PersistentFlags().StringVar(&cfg.IntwebURL, "url",
		"https://intweb.at.hive13.org/api/access",
		"URL of intweb server, including /api/access")
	rootCmd.PersistentFlags().StringVar(&cfg.IntwebDevice, "device",
		"", "intweb device name (required)")
	rootCmd.MarkPersistentFlagRequired("device")
	// This needs conversion to []byte:
	rootCmd.PersistentFlags().StringVar(&device_key, "key",
		"", "intweb device key (required)")
	rootCmd.MarkPersistentFlagRequired("key")
	rootCmd.PersistentFlags().StringVar(&cfg.IntwebItem, "item",
		"", "intweb item to attempt to access (required)")
	rootCmd.MarkPersistentFlagRequired("item")
	
	rootCmd.PersistentFlags().StringVar(&cfg.ListenAddr, "addr",
		":9000", "Address for HTTP server to listen on")

	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.BrokerAddr, "broker",
		"", "MQTT broker address, e.g. tcp://foobar.com:1883")
	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.TopicSensor, "topic-sensor",
		"door/sensor", "MQTT topic to publish door sensor readings")
	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.TopicBadge, "topic-badge",
		"door/badge", "MQTT topic to publish badge scans")
	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.Username, "mqtt-username",
		"", "Username for MQTT")
	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.Password, "mqtt-password",
		"", "Password for MQTT")
	rootCmd.PersistentFlags().StringVar(&cfg.Mqtt.ClientID, "mqtt-client-id",
		"", "Client ID for MQTT")
	
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v",
		false, "Enable more verbose logging")
}
