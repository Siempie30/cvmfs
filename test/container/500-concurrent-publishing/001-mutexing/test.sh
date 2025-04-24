docker exec  cvmfs-pub1 curl -X POST --data "{\"repo\":\"test.repo.org\"}" http://cvmfs-gw1:4929/api/v1/token-ring

echo "\n\n---Transaction 1: pub 1 (to gw1)"
docker exec  cvmfs-pub1 cvmfs_server transaction
docker exec  cvmfs-pub1 bash -c 'echo abc > /cvmfs/test.repo.org/testfile'

echo "\n\n---Publishing transaction 1"
docker exec  cvmfs-pub1 cvmfs_server publish

sleep 5

echo "\n\n---Transaction 2: pub 2 (to gw2)"
docker exec  cvmfs-pub2 cvmfs_server transaction
docker exec  cvmfs-pub2 bash -c 'echo def > /cvmfs/test.repo.org/testfile'

echo "\n\n---Publishing transaction 2"
docker exec  cvmfs-pub2 cvmfs_server publish

docker exec  cvmfs-gw1 cvmfs_server tag -l
docker exec  cvmfs-gw1 cat /cvmfs/test.repo.org/testfile

