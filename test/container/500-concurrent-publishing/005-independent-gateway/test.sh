UTIL_SCRIPT="$(dirname $0)/../test_util.sh"
. $UTIL_SCRIPT

# Post the token to the gateways
echo "---Handing out token to gateways"
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test1.repo.org\"}' http://cvmfs-gw1:4929/api/v1/hagroup/creation" || exit 1
execute_in_container cvmfs-gw1 "curl -s -X POST --data '{\"repo\":\"test2.repo.org\"}' http://cvmfs-gw1:4929/api/v1/hagroup/creation" || exit 2
execute_in_container cvmfs-gw2 "curl -s -X POST --data '{\"repo\":\"test3.repo.org\"}' http://cvmfs-gw2:4929/api/v1/hagroup/creation" || exit 3

# Start transactions
echo "\n---Starting transactions"
execute_in_container cvmfs-pub1 "cvmfs_server transaction test1.repo.org" || exit 4
execute_in_container cvmfs-pub2 "cvmfs_server transaction test2.repo.org" || exit 5
execute_in_container cvmfs-pub1 "cvmfs_server transaction test3.repo.org" || exit 6

# Apply changes
echo "\n---Applying changes"
execute_in_container cvmfs-pub1 "echo 'repo1-pub1' > /cvmfs/test1.repo.org/repo1-pub1" || exit 7
execute_in_container cvmfs-pub2 "echo 'repo2-pub2' > /cvmfs/test2.repo.org/repo2-pub2" || exit 8
execute_in_container cvmfs-pub1 "echo 'repo3-pub1' > /cvmfs/test3.repo.org/repo3-pub1" || exit 9

# Publish changes
echo "\n---Publishing changes"
execute_in_container cvmfs-pub1 "cvmfs_server publish test1.repo.org" || exit 10
execute_in_container cvmfs-pub2 "cvmfs_server publish test2.repo.org" || exit 11
execute_in_container cvmfs-pub1 "cvmfs_server publish test3.repo.org" || exit 12

# Verify changes for each repo on each gateway
echo "\n---Verifying changes"
execute_in_container cvmfs-gw1 "cvmfs_server mount test1.repo.org" || exit 13
execute_in_container cvmfs-gw1 "cvmfs_server mount test2.repo.org" || exit 14
execute_in_container cvmfs-gw2 "cvmfs_server mount test1.repo.org" || exit 15
execute_in_container cvmfs-gw2 "cvmfs_server mount test3.repo.org" || exit 16
execute_in_container cvmfs-gw3 "cvmfs_server mount test2.repo.org" || exit 17
execute_in_container cvmfs-gw3 "cvmfs_server mount test3.repo.org" || exit 18

execute_in_container cvmfs-gw1 "cat /cvmfs/test1.repo.org/repo1-pub1 | tee | grep \"repo1-pub1\"" || exit 19
execute_in_container cvmfs-gw1 "cat /cvmfs/test2.repo.org/repo2-pub2 | tee | grep \"repo2-pub2\"" || exit 20
execute_in_container cvmfs-gw2 "cat /cvmfs/test1.repo.org/repo1-pub1 | tee | grep \"repo1-pub1\"" || exit 21
execute_in_container cvmfs-gw2 "cat /cvmfs/test3.repo.org/repo3-pub1 | tee | grep \"repo3-pub1\"" || exit 22
execute_in_container cvmfs-gw3 "cat /cvmfs/test2.repo.org/repo2-pub2 | tee | grep \"repo2-pub2\"" || exit 23
execute_in_container cvmfs-gw3 "cat /cvmfs/test3.repo.org/repo3-pub1 | tee | grep \"repo3-pub1\"" || exit 24

# Wait for token for repo 3 to arrive at gateway 3 (the repo 3 token was the last to be posted)
for attempt in $(seq 1 20); do
    has_token=$(docker exec -it cvmfs-pub3 curl -s -X GET --data '{"repo":"test3.repo.org"}' http://cvmfs-gw3:4929/api/v1/hagroup | jq '.has_token')
    if [ "$has_token" = "true" ]; then
        echo "\n---Gateway 3 has token, continuing"
        break
    fi
    sleep 0.5
done

# Start transactions
echo "\n---Starting transactions"
execute_in_container cvmfs-pub3 "cvmfs_server transaction test1.repo.org" || exit 25
execute_in_container cvmfs-pub1 "cvmfs_server transaction test2.repo.org" || exit 26
execute_in_container cvmfs-pub3 "cvmfs_server transaction test3.repo.org" || exit 27

# Apply changes
echo "\n---Applying changes"
execute_in_container cvmfs-pub3 "echo 'repo1-pub3' > /cvmfs/test1.repo.org/repo1-pub3" || exit 28
execute_in_container cvmfs-pub1 "echo 'repo2-pub1' > /cvmfs/test2.repo.org/repo2-pub1" || exit 29
execute_in_container cvmfs-pub3 "echo 'repo3-pub3' > /cvmfs/test3.repo.org/repo3-pub3" || exit 30

# Publish changes
echo "\n---Publishing changes"
execute_in_container cvmfs-pub3 "cvmfs_server publish test1.repo.org" || exit 31
execute_in_container cvmfs-pub1 "cvmfs_server publish test2.repo.org" || exit 32
execute_in_container cvmfs-pub3 "cvmfs_server publish test3.repo.org" || exit 33

# Verify changes for each repo on each gateway
echo "\n---Verifying changes"
execute_in_container cvmfs-gw1 "cvmfs_server mount test1.repo.org" || exit 34
execute_in_container cvmfs-gw1 "cvmfs_server mount test2.repo.org" || exit 35
execute_in_container cvmfs-gw2 "cvmfs_server mount test1.repo.org" || exit 36
execute_in_container cvmfs-gw2 "cvmfs_server mount test3.repo.org" || exit 37
execute_in_container cvmfs-gw3 "cvmfs_server mount test2.repo.org" || exit 38
execute_in_container cvmfs-gw3 "cvmfs_server mount test3.repo.org" || exit 39

execute_in_container cvmfs-gw1 "cat /cvmfs/test1.repo.org/repo1-pub3 | tee | grep \"repo1-pub3\"" || exit 40
execute_in_container cvmfs-gw1 "cat /cvmfs/test2.repo.org/repo2-pub1 | tee | grep \"repo2-pub1\"" || exit 41
execute_in_container cvmfs-gw2 "cat /cvmfs/test1.repo.org/repo1-pub3 | tee | grep \"repo1-pub3\"" || exit 42
execute_in_container cvmfs-gw2 "cat /cvmfs/test3.repo.org/repo3-pub3 | tee | grep \"repo3-pub3\"" || exit 43
execute_in_container cvmfs-gw3 "cat /cvmfs/test2.repo.org/repo2-pub1 | tee | grep \"repo2-pub1\"" || exit 44
execute_in_container cvmfs-gw3 "cat /cvmfs/test3.repo.org/repo3-pub3 | tee | grep \"repo3-pub3\"" || exit 45