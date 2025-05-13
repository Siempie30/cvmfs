#!/bin/bash

# Configure and create the bucket
mc alias set local http://localhost:9000 minioadmin minioadmin123
mc rb --force local/mybucket
mc mb local/mybucket
mc anonymous set public local/mybucket