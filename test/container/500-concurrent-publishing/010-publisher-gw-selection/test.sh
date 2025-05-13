SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT

echo "---Stopping gateway 1"
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 1

echo "\n---Handing out token to gateway 2"
execute_in_container cvmfs-gw2 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw2:4929/api/v1/token-ring" || exit 2

echo "\n---Starting transaction on publisher"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 3

echo "\n---Verifying succesful lease on gateway 2"
OUTPUT=$(docker exec -it cvmfs-gw2 curl -X GET http://cvmfs-gw2:4929/api/v1/leases | jq '.data."test.repo.org/".hostname')
echo "$OUTPUT"
if [ "$OUTPUT" != "\"cvmfs-pub1\"" ]; then exit 4; fi

echo "\n---Writing changes"
execute_in_container cvmfs-pub1 "echo 'abc' > /cvmfs/test.repo.org/someFile" || exit 5

echo "\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 6
# NOTE: This test currently fails because the publish command makes use of the upstream storage value configured in server.conf
# This value is not yet updated when the default gateway (and therefore the gateway on which the lease is acquired) changes.
# Should the server.conf file be updated when the default gateway changes, or should the server.conf file read the token ring db
# to modify the upstream storage value passed to swissknife sync?

echo "\n---Verifying succesful changes on gateway 2"
execute_in_container cvmfs-gw2 "cvmfs_server mount test.repo.org" || exit 7
execute_in_container cvmfs-gw2 "cat /cvmfs/test.repo.org/someFile | tee | grep \"abc\"" || exit 8
