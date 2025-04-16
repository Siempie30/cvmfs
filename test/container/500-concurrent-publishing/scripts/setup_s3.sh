#!/bin/bash

# Configure alias for local MinIO server
mc alias set local http://localhost:9000 minioadmin minioadmin123

# Create a bucket (change 'mybucket' to whatever you like)
mc mb local/mybucket

# Set public or specific policy if needed
mc policy public local/mybucket