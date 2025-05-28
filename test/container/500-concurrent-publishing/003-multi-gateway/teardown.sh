docker rm -f cvmfs-pub1 && docker volume rm 500-concurrent-publishing_var_spool_pub1
docker rm -f cvmfs-pub2 && docker volume rm 500-concurrent-publishing_var_spool_pub2
docker rm -f cvmfs-pub3 && docker volume rm 500-concurrent-publishing_var_spool_pub3
docker rm -f cvmfs-pub4 && docker volume rm 500-concurrent-publishing_var_spool_pub4
docker rm -f cvmfs-pub5 && docker volume rm 500-concurrent-publishing_var_spool_pub5
docker rm -f cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_gw1
docker rm -f cvmfs-gw2 && docker volume rm 500-concurrent-publishing_var_spool_gw2
docker rm -f cvmfs-gw3 && docker volume rm 500-concurrent-publishing_var_spool_gw3
docker rm -f cvmfs-gw4 && docker volume rm 500-concurrent-publishing_var_spool_gw4
docker rm -f cvmfs-gw5 && docker volume rm 500-concurrent-publishing_var_spool_gw5
docker rm -f cvmfs-s3 && docker volume rm 500-concurrent-publishing_minio-data
SCRIPT_DIR=$(dirname $0)
rm -rf $SCRIPT_DIR/../keys/*