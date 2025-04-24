#!/bin/bash

# This script is used to copy the repo from the cache directory to the actual cvmfs repo directory on the container.
# It only copies the files that have been changed (or added) since the last build. This way, the build process is cached.

SOURCE_DIR="/home/sftnight/cvmfs-cache"
TARGET_DIR="/home/sftnight/cvmfs"

# Ensure source directory exists
if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "Source directory does not exist: $SOURCE_DIR"
  exit 1
fi

# Create target directory if it doesn't exist
mkdir -p "$TARGET_DIR"

# Use rsync to copy only changed files
rsync -av --update "$SOURCE_DIR"/ "$TARGET_DIR"/

# Explanation:
# -a: archive mode (preserves permissions, symbolic links, etc.)
# -v: verbose
# --update: skip files that are newer on the receiver (target)
