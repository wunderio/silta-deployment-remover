# Save key, authenticate and set compute zone
echo $GCLOUD_KEY_JSON > ${HOME}/gcloud-service-key.json
echo "Logging on to Google Cloud"
gcloud auth activate-service-account --key-file=${HOME}/gcloud-service-key.json --project $GCLOUD_PROJECT_NAME
gcloud config set compute/zone $GCLOUD_COMPUTE_ZONE

# Updates a kubeconfig file with appropriate credentials and endpoint information 
# to point kubectl at a specific cluster in Google Kubernetes Engine
gcloud container clusters get-credentials $GCLOUD_CLUSTER_NAME \
  --zone $GCLOUD_COMPUTE_ZONE \
  --project $GCLOUD_PROJECT_NAME

# Remove deployment
echo "Removing release $RELEASE_NAME"
helm delete --purge $RELEASE_NAME

# Remove jobs
kubectl delete job -l release=$RELEASE_NAME -n $NAMESPACE

# Remove PersistentVolumeClaim left over from StatefulSets
kubectl delete pvc -l release=$RELEASE_NAME -n $NAMESPACE
