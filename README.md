Hive13 RFID Access
==================

This contains code for Hive13's RFID & door access server, which:

- listens to a Wiegand-compatible badge reader for members scanning
  badges
- listens on an HTTP server for 'manual' open events (e.g. from a
  member wanting to trigger a door release remotely)
- communicates with [intweb](https://github.com/Hive13/HiveWeb) to
  verify that this badge has access
- caches allowed badges for a configurable amount of time to speed up
  process for repeat badges, but still requests access in the
  background so that intweb still logs an access
- triggers opening the door lock

For the older Arduino-based version of this that I did not write, see
[https://github.com/Hive13/hive-rfid-door-controller](https://github.com/Hive13/hive-rfid-door-controller).

This is written as a [Go](https://golang.org/) module.  As it is Go,
it requires compilation.

So far it has only been tested and run on Alpine Linux on a Raspberry
Pi.  In theory, it should run anywhere that Go can target, and that
has a working Linux GPIO driver (which
[gpiod](https://github.com/warthog618/gpiod) uses).

For more information, see
[https://wiki.hive13.org/view/RFID_Access#2701_Front_Door](https://wiki.hive13.org/view/RFID_Access).
Most information that is specific to Hive13's installations is kept
here.

It makes some sparse attempts at communication via the RFID reader
itself, which has an LED and a beeper:

- At successful startup, it will quickly beep several times.
- When it is running, it will blink the LED at regular intervals.
- When a badge is scanned, in addition to the reader's normal
  beeps and LED blinking (which cannot be controlled), it will:
  - Sound one long beep on a successful access
  - Sound two long beeps on a *denied* access
  - Sound three long beeps on any error that prevented it
    from even being able to query access with intweb (typically
    network problems or misconfiguration)

TODO:

- Document MQTT stuff.

Running
-------

See the below section for building the binary.  Once the binary is
built, a diskless Alpine install should suffice.

Run `./access.bin` to see its commandline options.  Many things can be
specified, some mandatory:

- Pin numbers for the badge reader
- Pin number to trigger the electronic strike
- URL for intweb
- Device, device key, and item being accessed on intweb
- Address for the HTTP server

HTTP API
--------

This runs an HTTP server which supports the below requests:

- POST to `/open_door`: Request that the door open, identically to as
  if a badge were scanned. Supply the badge number with form key
  `badge` set to the badge number, exactly as it appears in the
  database.
- GET to `/ping`: Return a 200 OK if the server's main loop is
  responding. Return an error in any other case.

Development
-----------

This requires that Go be installed, at least v1.13. Older versions may
work with some effort, but I haven't tried them.

Right now, this uses only pure Go dependencies, which brings a few
benefits:

- In order to build it, only Go needs to be installed - no other
  packages or libraries are required.
- In order to *run* it, no other dependencies are needed.
- Cross-compilation is very simple.

The following will cross-compile a static binary suitable to run on
Linux on the Raspbery Pi with no other requirements:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -o access.bin access/main/main.go
```

(`GOARCH=arm64` is also fine for all but the oldest Pis, I think.)

This should fetch all dependencies and produce a standalone binary,
`access.bin`.

This build has been tested on Linux (x86_64 and ARM) and on OS X.

In theory, the above should work on any architecture that runs Linux,
that Go can target (`GOARCH` may need changed), and that has GPIO
supported by the Linux GPIO character device.

Code Overview
-------------

- [access/main/main.go](./access/main/main.go) is the commandline
  entry point.
- [access/access.go](./access/access.go) is the top-level service that
  ties together everything below and is meant as a long-running
  process listening for requests.  The commandline produces a
  configuration and calls this.
- [intweb/intweb.go](./intweb/intweb.go) interfaces with intweb (which
  runs https://github.com/Hive13/HiveWeb) for access-specific
  functionality.
- [mqtt/mqtt.go](./mqtt/mqtt.go) provides a small struct to bundle
  together some MQTT parameters together. It works with the
  [paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang)
  library.
- [sensor/sensor.go](./sensor/sensor.go) provides a basic debouncing
  and monitoring function that is used for monitoring a door sensor
  that is potentially noisy.
- [wiegand/wiegand.go](./wiegand/wiegand.go) is a wrapper which turns
  badge access to a Go channel that reports all 26-bit Wiegand codes
  scanned.

Some test utilities are provided too:

- [test/wiegand/main.go](./test/wiegand/main.go) is a utility
  which tries to read Wiegand codes, and outputs them as it receives
  them. Note that pin numbers are hard-coded, and so they may require
  modification.
- [test/sensor/main.go](./test/sensor/main.go) is a utility which runs
  the internal debouncing/state-change routine on a given pin.

Deployment
----------

This is specific to Alpine Linux, however, it should adapt easily to
other distributions based on OpenRC, and would likely also adapt to
systemd or other startup systems.

See the [alpine](./alpine) directory.  The file
[alpine/init.d/door_access](./alpine/init.d/door_access) belongs in
`/etc/init.d` and is the OpenRC service for this (be sure it is
executable). The file
[alpine/conf.d/door_access](./alpine/conf.d/door_access) belongs in
`/etc/conf.d` and contains configuration for the service.  You will
need to edit this configuration for your own setup.

Test with `/etc/init.d/door_access start`.

Run the [appropriate
commands](https://wiki.alpinelinux.org/wiki/Alpine_Linux_Init_System)
to enable this service in OpenRC, e.g. `rc-update add door_access
default`, either reboot or run `rc-service door_access start`, and
`rc-status` to verify that it is running.

Logs will be written to `/var/log/door_access.log`.  You may wish to
set up `logrotate` in order to prevent the logs from growing too
large.

Older Code
----------

This was based on an older prototype written in Python and using a
Wiegand driver written in C (based on
[rpi-wiegand-reader](https://github.com/alperenguman/rpi-wiegand-reader.git)).
For this, see the `old` directory.

The C code may be useful for standalone testing, as it runs as a
commandline application which prints scanned badges to stdout as
comma-separated data.

Prior Go versions relied on go-rpio and on WiringPi. These
dependencies were removed in favor of gpiod.
