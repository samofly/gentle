#!/bin/bash

# This script dumps contents of TinyG flash.
# Usage:
#
# ./dump.sh PORT output.hex
# where PORT is something like /dev/ttyUSB0 or /dev/ttyACM1

set -ue

readonly PORT=$1
readonly OUTPUT=${2:-dump.hex}

echo "Connecting to TinyG bootloader..."
echo "See more info at https://github.com/synthetos/TinyG/wiki/TinyG-Updating-Firmware"

avrdude -p x192a3 -c avr109 -b 115200 -P $PORT -U flash:r:$OUTPUT:i
