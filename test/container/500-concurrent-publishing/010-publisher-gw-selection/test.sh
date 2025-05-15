SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT
DB_FILE="/var/spool/cvmfs/test.repo.org/tokenring.sqlite"

echo "---Stopping gateway 1"
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 1

echo "\n---Handing out token to gateway 2"
execute_in_container cvmfs-gw2 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw2:4929/api/v1/token-ring" || exit 2

# To make sure gateway 2 has time to update its token ring configuration by attempting to pass it to gw1
sleep 3

echo "\n---Starting transaction on publisher"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 3
OUTPUT=$(execute_in_container cvmfs-pub1 "sqlite3 $DB_FILE \"SELECT address FROM gateway WHERE current_gw=1;\"")
echo "$OUTPUT"
EXPECTED="http://cvmfs-gw2:4929/api/v1"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 4; fi

echo "\n---Verifying succesful lease on gateway 2"
OUTPUT=$(docker exec -it cvmfs-gw2 curl -X GET http://cvmfs-gw2:4929/api/v1/leases | jq '.data."test.repo.org/".hostname')
echo "$OUTPUT"
if [ "$OUTPUT" != "\"cvmfs-pub1\"" ]; then exit 4; fi

echo "\n---Writing changes"
execute_in_container cvmfs-pub1 "echo 'abc' > /cvmfs/test.repo.org/someFile" || exit 5

echo "\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish" || exit 6

echo "\n---Verifying succesful changes on gateway 2"
execute_in_container cvmfs-gw2 "cvmfs_server mount test.repo.org" || exit 7
execute_in_container cvmfs-gw2 "cat /cvmfs/test.repo.org/someFile | tee | grep \"abc\"" || exit 8

echo "\n---Verifying publisher token ring db is updated"
OUTPUT=$(execute_in_container cvmfs-pub1 "sqlite3 $DB_FILE \"SELECT address FROM gateway WHERE default_gw=1;\"")
echo "$OUTPUT"
EXPECTED="http://cvmfs-gw2:4929/api/v1"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 9; fi

