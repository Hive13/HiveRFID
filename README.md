Hive13 RFID Access
==================

This contains code for Hive13's RFID & door access server, which:

- listens to a Wiegand-compatible badge reader for members scanning
  badges
- listens on an HTTP server for 'manual' open events (e.g. from a
  member wanting to trigger a door release remotely)
- communicates with intweb to verify that this badge has access
- triggers opening the door lock

For more information, see
https://wiki.hive13.org/view/RFID_Access#2020_Temporary_Rework_for_Front_Door

This is written as a [Go](https://golang.org/) module. Currently it
requires a Raspberry Pi to run because it relies on
[WiringPi](http://wiringpi.com/).  As it is Go, it requires
compilation.

So far it has only been tested and run on Alpine Linux, but it should
build and run the same on any Raspberry Pi distribution.

Running
-------

See the below section for building the binary.  Once the binary is
built, you should require only the `wiringpi` packages, and a diskless
Alpine install should suffice.

Run `./access.bin` to see its commandline options.  Many things can be
specified, some mandatory:

- Pin numbers for the badge reader
- Pin number for the 
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

