remove_release () {
  # Remove deployment
  echo "REMOVER: Removing release $NAMESPACE/$RELEASE_NAME"
  helm delete -n "$NAMESPACE" "$RELEASE_NAME"
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME helm delete status: $?"

  # Remove jobs
  kubectl delete job -l release="$RELEASE_NAME" -n "$NAMESPACE"
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME job delete status: $?"

  # Remove PersistentVolumeClaim left over from StatefulSets
  kubectl delete pvc -l release="$RELEASE_NAME" -n "$NAMESPACE"
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME sts pvc delete status: $?"

  # Also remove PersistentVolumeClaims from Elasticsearch, that chart has different labels.
  kubectl delete pvc -l app="${RELEASE_NAME}-es" -n "$NAMESPACE"
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME es pvc delete status: $?"
  
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME release removed"
}

if [ ! `kubectl auth can-i delete pod` ] ; then
  # Save key, authenticate and set compute zone
  echo $GCLOUD_KEY_JSON > ${HOME}/gcloud-service-key.json
  echo "REMOVER: Logging on to Google Cloud"
  gcloud auth activate-service-account --key-file=${HOME}/gcloud-service-key.json --project $GCLOUD_PROJECT_NAME
  echo "REMOVER: gcloud auth status: $?"
  gcloud config set compute/zone $GCLOUD_COMPUTE_ZONE

  # Updates a kubeconfig file with appropriate credentials and endpoint information
  # to point kubectl at a specific cluster in Google Kubernetes Engine
  gcloud container clusters get-credentials $GCLOUD_CLUSTER_NAME \
    --zone $GCLOUD_COMPUTE_ZONE \
    --project $GCLOUD_PROJECT_NAME
  echo "REMOVER: gcloud container clusters status: $?"
fi

# Legacy remover
remove_release

# Select release name label based on branchName in configmapdata
RELEASE_NAMES=$(kubectl -n "${NAMESPACE}" get cm -o json | jq -r '.items | map(select(.data.branchName == env.BRANCH_NAME )) | .[].metadata.labels.release')

# Iterate over release names and remove them
for RELEASE_NAME in $RELEASE_NAMES
do
	remove_release
done
