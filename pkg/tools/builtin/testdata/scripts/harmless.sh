#!/bin/bash

# This is a harmless test script
echo "Hello from harmless script"
echo "Current directory: $(pwd)"
echo "Script arguments: $@"
echo "Environment test: $TEST_VAR"

# Sleep for a short time if requested
if [ "$1" = "sleep" ]; then
    echo "Sleeping for 2 seconds..."
    sleep 2
    echo "Done sleeping"
fi

# Return different exit codes based on argument
if [ "$1" = "fail" ]; then
    echo "This script is designed to fail"
    exit 1
fi

echo "Script completed successfully"
exit 0
