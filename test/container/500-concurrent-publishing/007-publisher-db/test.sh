SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT
DB_FILE="/var/spool/cvmfs/test.repo.org/tokenring.sqlite"

echo "---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 1

echo "\n---Starting transaction"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 2

echo "\n---Verifying publisher db"
OUTPUT=$(execute_in_container cvmfs-pub1 "sqlite3 $DB_FILE \"SELECT * FROM gateway;\"")
echo "$OUTPUT"
EXPECTED="http://cvmfs-gw1:4929/api/v1|0|1" # Address, status is 0, default is 1
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 3; fi

echo "\n---Aborting transaction"
execute_in_container cvmfs-pub1 "cvmfs_server abort" || exit 4

echo "\n---Setting up gateway 2"
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw2/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw2/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw1/repo.json
docker exec -it cvmfs-gw2 /scripts/gateway_mkfs.sh -E test.repo.org || exit 5
docker exec -it cvmfs-gw2 systemctl start cvmfs-gateway

echo "\n---Waiting for token to return to gateway 1"
sleep 6

echo "\n---Starting transaction"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 6

echo "\n---Verifying publisher db"
OUTPUT=$(execute_in_container cvmfs-pub1 "sqlite3 $DB_FILE \"SELECT * FROM gateway;\"")
echo "$OUTPUT"
EXPECTED="http://cvmfs-gw1:4929/api/v1|0|1
http://cvmfs-gw2:4929/api/v1|0|0"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 7; fi

echo "\n---Aborting transaction"
execute_in_container cvmfs-pub1 "cvmfs_server abort" || exit 8

echo "\n---Stopping gateway 2"
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 9

sleep 2

echo "\n---Starting transaction"
execute_in_container cvmfs-pub1 "cvmfs_server transaction" || exit 10

echo "\n---Verifying publisher db"
OUTPUT=$(execute_in_container cvmfs-pub1 "sqlite3 $DB_FILE \"SELECT * FROM gateway;\"")
echo "$OUTPUT"
EXPECTED="http://cvmfs-gw1:4929/api/v1|0|1
http://cvmfs-gw2:4929/api/v1|3|0" # Status is now 3 (down)
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 11; fi
