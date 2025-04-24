# Test of concurrent publishing


## Running the test
1. Create tar of the cvmfs repo
```sh
tar -czvf repo.tar.gz ../../..
```
2. Build and start the containers
```sh
docker-compose up --build -d
```
3. Setup CVMFS
```sh
./setup.sh
```
4. Run the test
```sh
./test.sh
```

## Resetting the containers
This stops and removes the created containers and their volumes
```sh
./teardown.sh
```

