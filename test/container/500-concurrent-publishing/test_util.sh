#!/bin/bash

# This function calls the specified command in the specified container, and returns the exit code of the command.
execute_in_container() {
    local container_name="$1"
    local command="$2"

    docker exec "$container_name" bash -c "$command; exit \$?"   
    return $?
}