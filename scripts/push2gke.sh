# dir
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

# push2gke
$SCRIPT_DIR/../db/redis/push2gke_artifact_redis.sh
