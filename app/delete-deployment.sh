remove_release () {
  # Delete post-release job first
  kubectl delete job "${RELEASE_NAME}-post-release" -n "$NAMESPACE"
  echo "REMOVER: $NAMESPACE/$RELEASE_NAME post-release delete status: $?"

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

# Legacy remover
remove_release

# Select release name label based on branchName in configmapdata
RELEASE_NAMES=$(kubectl -n "${NAMESPACE}" get cm -o json | jq -r '.items | map(select(.data.branchName == env.BRANCH_NAME )) | .[].metadata.labels.release')

# Iterate over release names and remove them
for RELEASE_NAME in $RELEASE_NAMES
do
	remove_release
done
