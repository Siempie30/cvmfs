#!/bin/bash

# Configure and create the bucket
mc alias set local http://cvmfs-s3:9000 minioadmin minioadmin123
mc mb local/mybucket
mc anonymous set public local/mybucket