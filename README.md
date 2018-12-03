# About project

This container exposes GIT webhooks that will trigger kubernetes cluster helm deployment removal on branch deletion. Once webhook request is received, it will log on to Google Kubernetes Cluster and remove the deployment.

# Configuration

1. You have to add a custom application to your Github organisation (https://github.com/organizations/<org name>/settings/apps/new). Minimal configuration needs `Webhook URL` and `Webhook secret (optional)` defined. Don't forget to enable this application in your organisation (https://github.com/organizations/<org name>/settings/apps/<app name>/installations). URL address is `<host>/webhooks` (port 80)

2. Redis server for task queue

3. Provide following environment variables for the container

  - WEBHOOKS_SECRET
  - REDIS_ADDR
  - GCLOUD_KEY_JSON
  - GCLOUD_PROJECT_NAME
  - GCLOUD_COMPUTE_ZONE
  - GCLOUD_CLUSTER_NAME


