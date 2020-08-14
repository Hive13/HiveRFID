Hive13 RFID Access
==================

This contains code for Hive13's RFID & door access server, which:

- listens to a Wiegand-compatible badge reader for members scanning
  badges
- listens on an HTTP server for 'manual' open events (e.g. from a
  member wanting to trigger a door release remotely)
- communicates with [intweb](https://github.com/Hive13/HiveWeb) to
  verify that this badge has access
- triggers opening the door lock

For the older Arduino-based version of this that I did not write, see
[https://github.com/Hive13/hive-rfid-door-controller](https://github.com/Hive13/hive-rfid-door-controller).

This is written as a [Go](https://golang.org/) module. Currently it
requires a Raspberry Pi to run because it relies on
[WiringPi](http://wiringpi.com/).  As it is Go, it requires
compilation.

So far it has only been tested and run on Alpine Linux, but it should
build and run the same on any Raspberry Pi distribution.

For more information, see
[https://wiki.hive13.org/view/RFID_Access#2929_Spring_Grove_-_Rework_for_Front_Door](https://wiki.hive13.org/view/RFID_Access#2929_Spring_Grove_-_Rework_for_Front_Door).
Most information that is specific to Hive13's installations is kept
here.

Running
-------

See the below section for building the binary.  Once the binary is
built, you should require only the `wiringpi` packages, and a diskless
Alpine install should suffice.

Run `./access.bin` to see its commandline options.  Many things can be
specified, some mandatory:

- Pin numbers for the badge reader
- Pin number to trigger the electronic strike
- URL for intweb
- Device, device key, and item being accessed on intweb
- Address for the HTTP server

Development
-----------

This requires that Go be installed, at least v1.13. Older versions may
work with some effort, but I haven't tried them. It also requires the
development libraries for WiringPi.

On Alpine Linux I am using the following packages: `go gcc libc-dev
wiringpi wiringpi-dev`. You may need to set up a [persistent
installation](https://wiki.alpinelinux.org/wiki/Classic_install_or_sys_mode_on_Raspberry_Pi)
rather than a diskless one in order to handle the extra space
required.

Run the following:

```bash
go build -o access.bin access/main/main.go
```

This should fetch all dependencies and produce a binary, `access.bin`.

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
- [wiegand/wiegand.go](./wiegand/wiegand.go) is a wrapper which turns
  badge access to a Go channel that reports all 26-bit Wiegand codes
  scanned.
- [wiegand/wiegand_c.go](./wiegand/wiegand_c.go) is the low-level code
  for accessing a Wiegand-compatible badge reader.  It relies on
  [cgo](https://golang.org/cmd/cgo/) and
  [WiringPi](http://wiringpi.com/).

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
