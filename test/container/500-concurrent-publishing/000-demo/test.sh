SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT

echo "---Passing token to gateway 1"
execute_in_container cvmfs-pub1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

echo "\n---Transaction 1: pub 1, unimportant change"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 2
execute_in_container cvmfs-pub1 "echo abc > /cvmfs/test.repo.org/testfile"

echo "\n---Publishing transaction 1"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 3

echo "\n---Transaction 2: pub 1, important change"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 4
execute_in_container cvmfs-pub1 "echo important_change > /cvmfs/test.repo.org/testfile" || exit 5

echo "\n\n---Transaction 3: pub 2, unimportant change"
execute_in_container cvmfs-pub2 "cvmfs_server transaction -t 5000" &
sleep 1

echo "\n\n---Publishing transaction 2"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 7
wait $(jobs -p) 
sleep 1

execute_in_container cvmfs-pub2 "echo unrelated_change > /cvmfs/test.repo.org/another_file" || exit 8

echo "\n\n---Publishing transaction 3"
execute_in_container cvmfs-pub2 "cvmfs_server publish" || exit 9

docker exec cvmfs-gw1 cvmfs_server tag -l
docker exec cvmfs-gw1 cat /cvmfs/test.repo.org/another_file
docker exec cvmfs-gw1 cat /cvmfs/test.repo.org/testfile
docker exec cvmfs-gw1 cat /cvmfs/test.repo.org/testfile | tee | grep important_change ||  echo -e ERROR >&2

