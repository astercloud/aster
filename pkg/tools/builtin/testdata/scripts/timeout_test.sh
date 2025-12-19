#!/bin/bash

# This script simulates a long-running process for timeout testing
echo "Starting long-running process..."
echo "PID: $$"

# Sleep for the specified number of seconds (default 10)
DURATION=${1:-10}
echo "Will sleep for $DURATION seconds"

for i in $(seq 1 $DURATION); do
    echo "Progress: $i/$DURATION"
    sleep 1
done

echo "Process completed successfully"
exit 0
