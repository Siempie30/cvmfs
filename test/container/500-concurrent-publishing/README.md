# Test of concurrent publishing


## Running a test
1. Build and start the containers
```sh
docker-compose up --build -d
```
2. Setup the test. Replace `<NNN>` and `<TESTNAME>` with the test number and name
```sh
./<NNN>-<TESTNAME>/setup.sh
```
3. Run the test. Replace `<NNN>` and `<TESTNAME>` with the test number and name
```sh
./<NNN>-<TESTNAME>/test.sh
```

## Resetting the containers
This stops and removes the created containers and their volumes. Make sure to do this before rerunning a test. Replace `<NNN>` and `<TESTNAME>` with the test number and name
```sh
./<NNN>-<TESTNAME>/teardown.sh
```

