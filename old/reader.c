// Based on:
// https://github.com/alperenguman/rpi-wiegand-reader.git
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <wiringPi.h>
#include <time.h>
#include <unistd.h>
#include <memory.h>

#define PIN_0 0 // GPIO Pin 17 | Green cable | Data0
#define PIN_1 1 // GPIO Pin 18 | White cable | Data1
#define PIN_SOUND 25 // GPIO Pin 26 | Yellow cable | Sound

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
    wiringPiSetup() ;
    pinMode(d0pin, INPUT);
    pinMode(d1pin, INPUT);
    pinMode(PIN_SOUND, OUTPUT);

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

void beep(int msec, int times){
    for (unsigned int i = 0; i < times; i++) {
        digitalWrite(PIN_SOUND,  LOW);
        delay(msec);
        digitalWrite(PIN_SOUND, HIGH);
        delay(msec/2);
    }
}

int main(void) {
    unsigned int i, j;
    unsigned int num_bits;
    bool data[100];
    bool even_check = 0;
    bool odd_check = 0;
    unsigned long val = 0;
    unsigned long mask = 1;

    init(PIN_0, PIN_1);

    while(1) {
        num_bits = pending_bit_count();
        if (num_bits == 0) {
            usleep(5000);
        } else {
            
            num_bits = copy_data((void *)data, 100);
            
            printf("%lu,", (unsigned long)time(NULL));
            printf("%u,", num_bits);
            for (i = 0; i < num_bits; i++) {
                printf("%u", data[i]);
            }
            printf(",");

            if (num_bits == 26) {

                even_check = 0;
                odd_check = 0;
                for (j = 0; j < 13; ++j) {
                    even_check = even_check ^ (data[j]);
                    odd_check  = odd_check  ^ (data[j + 13]);
                }

                if (!even_check && odd_check) {
                    val = 0;
                    mask = 1;
                    for (j = 24; j > 0; --j) {
                        if (data[j]) {
                            val |= mask;
                        }
                        mask <<= 1;
                    }
                    printf("OK,%lu\n", val);
                } else {
                    printf("ERROR,parity\n");
                }
            } else {
                printf("ERROR,wrong bit count\n");
            }

            fflush(stdout);
            
            beep(200, 1);
        }
    }

    return 0;
}
