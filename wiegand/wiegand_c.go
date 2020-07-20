package wiegand

// #cgo LDFLAGS: -lpthread -lwiringPi -lrt
/*
// Based on:
// https://github.com/alperenguman/rpi-wiegand-reader.git
#include <stdbool.h>
#include <wiringPi.h>
#include <time.h>
#include <memory.h>

#define MAX_WIEGAND_BITS 32
#define READER_TIMEOUT 3000000

static bool _isr_data[MAX_WIEGAND_BITS];
static unsigned long _isr_bit_count;
static struct timespec _isr_bit_time;

void d0_fall_isr(void) {
    if (_isr_bit_count < MAX_WIEGAND_BITS) {
        _isr_bit_count++;
    }
    clock_gettime(CLOCK_MONOTONIC, &_isr_bit_time);
}

void d1_fall_isr(void) {
    if (_isr_bit_count < MAX_WIEGAND_BITS) {
        _isr_data[_isr_bit_count] = 1;
        _isr_bit_count++;
    }
    clock_gettime(CLOCK_MONOTONIC, &_isr_bit_time);
}

void init(int d0pin, int d1pin) {
    wiringPiSetupGpio();
    pinMode(d0pin, INPUT);
    pinMode(d1pin, INPUT);
    wiringPiISR(d0pin, INT_EDGE_FALLING, d0_fall_isr);
    wiringPiISR(d1pin, INT_EDGE_FALLING, d1_fall_isr);
}

void reset() {
    memset((void *)_isr_data, 0, MAX_WIEGAND_BITS);
    _isr_bit_count = 0;
}

unsigned int pending_bit_count() {
    struct timespec now, delta;
    clock_gettime(CLOCK_MONOTONIC, &now);
    delta.tv_sec = now.tv_sec - _isr_bit_time.tv_sec;
    delta.tv_nsec = now.tv_nsec - _isr_bit_time.tv_nsec;

    if ((delta.tv_sec > 1) || (delta.tv_nsec > READER_TIMEOUT)) {
        return _isr_bit_count;
    }

    return 0;
}

unsigned int copy_data(void* data, int max_len) {
    if (pending_bit_count() > 0) {
        unsigned long count = _isr_bit_count;
        memcpy(data, (void *)_isr_data, ((count > max_len) ? max_len : count));

        reset();
        return count;
    }
    return 0;
}
*/
import "C"
import "unsafe"

func badgeCheckRaw(data *[data_len]byte) (uint, bool) {
	n := C.pending_bit_count()
	if n == 0 {
		return 0, false
	}

	return uint(C.copy_data(unsafe.Pointer(data), data_len)), true
}

// init initializes WiringPi, the interrupts, and pins.
//
// d0_pin and d1_pin are the WiringPi pin numbers for Wiegand D0 and
// Wiegand D1 pins, respectively.
//
// If this fails, it will simply exit the program due to how
// wiringPiSetup works.
func initWiegand(d0_pin int, d1_pin int) {
	C.init(C.int(d0_pin), C.int(d1_pin))
}
