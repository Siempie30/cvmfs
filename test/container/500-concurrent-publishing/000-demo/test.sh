docker exec  cvmfs-pub1 curl -X POST --data "{\"repo\":\"test.repo.org\"}" http://cvmfs-gw1:4929/api/v1/token-ring

echo "\n\n---Transaction 1: pub 1, unimportant change"
docker exec  cvmfs-pub1 cvmfs_server transaction
docker exec  cvmfs-pub1 bash -c 'echo abc > /cvmfs/test.repo.org/testfile'
echo "\n\n---Publishing transaction 1"
docker exec  cvmfs-pub1 cvmfs_server publish

echo "\n\n---Transaction 2: pub 1, important change"
docker exec  cvmfs-pub1 cvmfs_server transaction
docker exec  cvmfs-pub1 bash -c 'echo important_change > /cvmfs/test.repo.org/testfile'

echo "\n\n---Transaction 3: pub 2, unimportant change"
docker exec  cvmfs-pub2 cvmfs_server transaction -t 5000 &
sleep 1

echo "\n\n---Publishing transaction 2"
docker exec  cvmfs-pub1 cvmfs_server publish
wait $(jobs -p) 
sleep 1

docker exec  cvmfs-pub2 bash -c 'echo unrelated_change > /cvmfs/test.repo.org/another_file'
echo "\n\n---Publishing transaction 3"
docker exec  cvmfs-pub2 cvmfs_server publish

docker exec  cvmfs-gw1 cvmfs_server tag -l
docker exec  cvmfs-gw1 cat /cvmfs/test.repo.org/another_file
docker exec  cvmfs-gw1 cat /cvmfs/test.repo.org/testfile
docker exec  cvmfs-gw1 cat /cvmfs/test.repo.org/testfile | tee | grep important_change ||  echo -e ERROR >&2; exit 1

