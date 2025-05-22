UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

execute_in_container cvmfs-pub1 "curl -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

echo "---Transaction 1: pub 1 (to gw1)"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 2
execute_in_container cvmfs-pub1 "bash -c 'echo abc > /cvmfs/test.repo.org/testfile'" || exit 3

echo "\n---Publishing transaction 1"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 4

echo "\n\n-- Transaction 2: pub 2 (to gw2) (should fail)"
execute_in_container cvmfs-pub2 "cvmfs_server transaction" && exit 5 # Only exit if the command succeeds

sleep 5

echo "\n---Transaction 3: pub 2 (to gw2)"
execute_in_container cvmfs-pub2 "cvmfs_server transaction" || exit 6
execute_in_container cvmfs-pub2 "bash -c 'echo def > /cvmfs/test.repo.org/testfile'" || exit 7

echo "\n---Publishing transaction 3"
execute_in_container cvmfs-pub2 "cvmfs_server publish" || exit 8

execute_in_container cvmfs-gw1 "cvmfs_server mount; cat /cvmfs/test.repo.org/testfile" || exit 9
docker exec  cvmfs-gw1 cat /cvmfs/test.repo.org/testfile | tee | grep def ||  exit 10
