# Test of concurrent publishing


## Running the test
1. Build and start the containers
```sh
docker-compose up --build -d
```
2. Setup CVMFS
```sh
./setup.sh
```
3. Run the test
```sh
./test.sh
```

## Resetting the containers
This stops and removes the created containers and their volumes
```sh
./teardown.sh
```

