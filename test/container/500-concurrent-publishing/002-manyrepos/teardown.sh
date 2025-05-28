docker rm -f cvmfs-pub1 && docker volume rm 500-concurrent-publishing_var_spool_pub1
docker rm -f cvmfs-gw1 && docker volume rm 500-concurrent-publishing_var_spool_gw1
docker rm -f cvmfs-s3 && docker volume rm 500-concurrent-publishing_minio-data
SCRIPT_DIR=$(dirname $0)
rm -rf $SCRIPT_DIR/../keys/*