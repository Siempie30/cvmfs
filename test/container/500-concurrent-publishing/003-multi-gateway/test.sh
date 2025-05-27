UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

# Post the token to the first gateway
echo "---Handing out token to gateway 1"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test.repo.org\"}' http://cvmfs-gw1:4929/api/v1/hagroup/creation" || exit 1

echo "\n---Executing transactions"
for i in $(seq 1 5); do
    echo "----Gateway $i"
    # Start the transaction
    execute_in_container cvmfs-pub${i} "cvmfs_server transaction test.repo.org" || exit 2
    
    # Write changes
    execute_in_container cvmfs-pub${i} "echo '${i}' > /cvmfs/test.repo.org/testFile${i}" || exit 3
    
    # Publish changes
    execute_in_container cvmfs-pub${i} "cvmfs_server publish test.repo.org" || exit 4
    
    # Wait for token to be handed to next gateway
    next_i=$(($i + 1))
    if [ $next_i -gt 5 ]; then
        break
    fi
    for attempt in $(seq 1 20); do
        has_token=$(docker exec -it cvmfs-pub${next_i} curl -s -X GET --data '{"repo":"test.repo.org"}' http://cvmfs-gw${next_i}:4929/api/v1/hagroup | jq '.has_token')
        if [ "$has_token" = "true" ]; then
            echo "----Gateway $next_i has token, continuing loop"
            break
        fi
        sleep 0.5
    done
done

echo "\n---Checking content"
for i in $(seq 1 5); do
    for j in $(seq 1 5); do
        # Check if the file exists
        execute_in_container cvmfs-gw${i} "cvmfs_server mount test.repo.org; ls /cvmfs/test.repo.org/testFile${j}" || exit 5
        
        # Check if the content is correct
        execute_in_container cvmfs-gw${i} "cat /cvmfs/test.repo.org/testFile${j} | tee | grep \"${j}\"" || exit 6
    done
done
