package main

// Commandline for RFID/door access server. This turns arguments to a
// configuration, but is not responsible for any of the actual logic.

import (
	"log"
	"github.com/spf13/cobra"

	"hive13/rfid/access"
)

var cfg *access.Config
var device_key string

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
		// fine but Cobra can't read bytestrings directly):
		cfg.IntwebDeviceKey = []byte(device_key)

		log.Printf("%+v", cfg)

		// We have a configuration. Go run the server.
		access.Run(cfg)
	},
}

func init() {
	cfg = &access.Config{}

	rootCmd.PersistentFlags().IntVar(&cfg.PinD0, "d0", 17,
		"BCM/GPIO pin number for badge reader's Wiegand D0 pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinD1, "d1", 18,
		"BCM/GPIO pin number for badge reader's Wiegand D1 pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinBeeper, "beeper", 26,
		"BCM/GPIO pin number for badge reader's beeper pin")
	rootCmd.PersistentFlags().IntVar(&cfg.PinLED, "led", 16,
		"BCM/GPIO pin number for badge reader's beeper pin")

	rootCmd.PersistentFlags().IntVar(&cfg.PinLock, "lock", -1,
		"BCM/GPIO pin number to control door lock/latch")
	
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

	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v",
		false, "Enable more verbose logging")
}
