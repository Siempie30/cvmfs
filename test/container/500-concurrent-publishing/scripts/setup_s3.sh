#!/bin/bash

echo "Configuring alias for local MinIO server"
mc alias set local http://localhost:9000 minioadmin minioadmin123

echo "Set policy to public"
mc policy public local/mybucket