#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$(dirname "$(realpath "$0")")"

# Change to the script's directory
cd "$SCRIPT_DIR" || exit 1

#sudo docker-compose build

# Iterate through subdirectories starting with three digits
for dir in [0-9][0-9][0-9]*/; do
    if [[ -d "$dir" ]]; then
        echo "Executing test: $dir"
        sudo docker-compose up -d
        sleep 5
        
        # Execute setup.sh if it exists
        if [[ -f "$dir/setup.sh" ]]; then
            echo "Running setup.sh in $dir"
            bash "$dir/setup.sh" || exit 1
        else 
            echo "No setup.sh found in $dir"
            exit 1
        fi

        # Execute test.sh if it exists
        if [[ -f "$dir/test.sh" ]]; then
            echo "Running test.sh in $dir"
            bash "$dir/test.sh" || exit $?
            echo "HERE"
        else
            echo "No test.sh found in $dir"
            exit 1
        fi

        # Execute teardown.sh if it exists
        if [[ -f "$dir/teardown.sh" ]]; then
            echo "Running teardown.sh in $dir"
            bash "$dir/teardown.sh" || exit 1
        else
            echo "No teardown.sh found in $dir"
            exit 1
        fi
    fi
done