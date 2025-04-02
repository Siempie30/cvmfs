# Containerized token ring testing
This setup is intended for experimenting and testing with the addition of token ring, in the context of adding support for multiple gateways.

## Prerequisites
- S3 container usable as backend storage
- `s3.conf` file present in this directory (see [CVMFS documentation](https://cvmfs.readthedocs.io/en/stable/cpt-repo.html#s3-compatible-storage-systems))

## Running the containers
1. Build the gateway containers
```sh
sudo docker-compose build
```
2. Start the gateway containers
```sh
sudo docker-compose up
```