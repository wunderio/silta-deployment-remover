# About project

This container exposes GIT webhooks that will trigger kubernetes cluster helm deployment removal on branch deletion. Once webhook request is received, it will log on to Google Kubernetes Cluster and remove the deployment.

# Deployment requirements 
1. Redis server for task queue

2. Provide following environment variables for the container

  - WEBHOOKS_SECRET
  - REDIS_HOST
  - REDIS_PASSWORD
  - GCLOUD_KEY_JSON
  - GCLOUD_PROJECT_NAME
  - GCLOUD_COMPUTE_ZONE
  - GCLOUD_CLUSTER_NAME


# Configuration

1. You have to add a custom application to your Github organisation (https://github.com/organizations/<org name>/settings/apps/new). Minimal configuration needs `Webhook URL` and `Webhook secret (optional)` defined. 

2. Aplication permissions & webhooks section, add `Repository contents` r/o permission and check `Delete (Branch or tag deleted.)` option in `
Subscribe to events` section.

3. nable this application in your organisation (https://github.com/organizations/<org name>/settings/apps/<app name>/installations). URL address is `<host>/webhooks` (port 80)

