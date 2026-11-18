#!/bin/sh

PROJECT_ID=kyle-server-402706
REPOSITORY=kyle-registry
LOCATION=me-west1
IMAGE=redis-tcp-app-server
# The version is an argument: this read TAG=0.0.1 while production ran
# 0.0.3 (2026-11-18), so a release meant editing the script first.
TAG=${1:?usage: push2gke_artifact.sh <version>}
TAG_LATEST=latest

# dir
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
DOCKERFILE_PATH=$SCRIPT_DIR/Dockerfile
CONTEXT_PATH=$SCRIPT_DIR

set -e

# The previous image is not deleted here. This used to untag the version
# and `latest` and then delete the newest digest in the package. On
# 2026-11-18 that digest was the attestation manifest inside 0.0.3, the image
# production was running, so the step would have broken pulling the live
# image and left nothing to roll back to. Old versions are aged out by the
# repository's cleanup policy instead (TAGGED older than 30 days, keeping
# the newest 12), the same as kyle-server's push scripts.

IMAGE_TAG=$LOCATION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE:$TAG
# --provenance=false: by default buildx publishes an OCI index with the image
# and an attestation manifest, and Artifact Registry lists the children as
# untagged versions of their own; deleting one breaks `docker pull` of the
# tagged image. One manifest per release keeps that from being possible.
# PROGRAM_VER is the version the binary reports; it was fixed at 0.0.1.
docker buildx build --provenance=false --no-cache --platform linux/amd64 --build-arg=PROGRAM_VER=$TAG --load -t $IMAGE_TAG -f $DOCKERFILE_PATH $CONTEXT_PATH
docker push $IMAGE_TAG

IMAGE_TAG_LATEST=$LOCATION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY/$IMAGE:$TAG_LATEST
docker tag $IMAGE_TAG $IMAGE_TAG_LATEST
docker push $IMAGE_TAG_LATEST

# # git tag -a $1 -m "add tag for $1"
# # git push origin main --tags

# # gcloud container clusters get-credentials kyle-ingress-standard-cluster-1 --zone me-west1-a --project kyle-server-402706
