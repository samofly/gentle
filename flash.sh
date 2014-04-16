#!/bin/bash

# This script writes the input hex file to the TinyG flash memory
# Usage:
#
# ./dump.sh PORT input.hex
# where PORT is something like /dev/ttyUSB0 or /dev/ttyACM1
# See more at https://github.com/synthetos/TinyG/wiki/TinyG-Updating-Firmware

set -ue

readonly PORT=$1
readonly INPUT=$2

echo "Connecting to TinyG bootloader..."
echo "See more info at https://github.com/synthetos/TinyG/wiki/TinyG-Updating-Firmware"

avrdude -p x192a3 -c avr109 -b 115200 -P $PORT -e -U flash:w:$INPUT
