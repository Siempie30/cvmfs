SCRIPT_DIR=$(dirname $0)
UTIL_SCRIPT="$SCRIPT_DIR/../test_util.sh"
. $UTIL_SCRIPT
DB_FILE="/var/lib/cvmfs-gateway/gw.db"

echo "---Verifying initial token ring tables"
EXPECTED="http://cvmfs-gw1:4929/api/v1|test.repo.org|0
http://cvmfs-gw2:4929/api/v1|test.repo.org|0"
OUTPUT=$(execute_in_container cvmfs-gw1 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 1; fi
OUTPUT=$(execute_in_container cvmfs-gw2 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 2; fi

echo "\n---Creating third gateway"
cp $SCRIPT_DIR/user.json $SCRIPT_DIR/../config/gw3/user.json
cp $SCRIPT_DIR/token_ring.json $SCRIPT_DIR/../config/gw3/token_ring.json
cp $SCRIPT_DIR/repo.json $SCRIPT_DIR/../config/gw3/repo.json
docker exec -it cvmfs-gw3 /scripts/gateway_mkfs.sh -E test.repo.org || exit 3
docker exec -it cvmfs-gw3 systemctl start cvmfs-gateway

echo "\n---Verifying updated token ring tables"
EXPECTED="http://cvmfs-gw1:4929/api/v1|test.repo.org|0
http://cvmfs-gw2:4929/api/v1|test.repo.org|0
http://cvmfs-gw3:4929/api/v1|test.repo.org|0"
OUTPUT=$(execute_in_container cvmfs-gw1 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 4; fi
OUTPUT=$(execute_in_container cvmfs-gw2 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 5; fi
OUTPUT=$(execute_in_container cvmfs-gw3 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 6; fi

echo "\n---Passing token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/token-ring" || exit 7

echo "\n---Removing gateway 2"
execute_in_container cvmfs-gw2 "systemctl stop cvmfs-gateway" || exit 8

sleep 3

echo "\n---Verifying gateway 3 has token"
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 9; fi

echo "\n---Verifying updated token ring tables"
EXPECTED="http://cvmfs-gw1:4929/api/v1|test.repo.org|0
http://cvmfs-gw2:4929/api/v1|test.repo.org|3
http://cvmfs-gw3:4929/api/v1|test.repo.org|0"
OUTPUT=$(execute_in_container cvmfs-gw1 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 10; fi
OUTPUT=$(execute_in_container cvmfs-gw3 "sqlite3 $DB_FILE \"SELECT * FROM TokenRing;\"")
echo "$OUTPUT"
if [ "$OUTPUT" != "$EXPECTED" ]; then exit 11; fi

sleep 3

echo "\n---Verifying gateway 1 has token and gateway 3 doesn't"
has_token=$(docker exec -it cvmfs-gw1 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw1:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 12; fi
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 13; fi

echo "\n---Stopping gateway 1"
execute_in_container cvmfs-gw1 "systemctl stop cvmfs-gateway" || exit 13

echo "\n---Verifying gateway 3 has token during cycle time (16 seconds)"

sleep 4
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 14; fi

sleep 4
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 15; fi

sleep 4
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "false" ]; then exit 16; fi

sleep 4
has_token=$(docker exec -it cvmfs-gw3 curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw3:4929/api/v1/token-ring | jq '.has_token')
if [ "$has_token" != "true" ]; then exit 17; fi