# About project

This container exposes GIT webhooks that will trigger kubernetes cluster helm deployment removal on branch deletion. Once webhook request is received, it will log on to Google Kubernetes Cluster and remove the deployment.

Compatibility:
- Github webhooks ([webhook documentation](https://docs.github.com/en/developers/webhooks-and-events/webhooks/webhook-events-and-payloads#push))
  - Payload URL: `<domain>/webhooks`
  - Content type: `application/x-www-form-urlencoded`
  - Secret: `<same as WEBHOOKS_SECRET>`
  - Individual events: `Branch or tag deletion`
- Azure Repos service Hooks ([webhook documentation](https://docs.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops)):
  - Service: `Web Hooks`
  - Trigger on this type of event: `Code pushed`
  - URL: `<domain>/webhooks`
  - Basic authentication username: `<any>`
  - Basic authentication password: `<same as WEBHOOKS_SECRET>`
- (untested) GitLab ([webhook documentation](https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#push-events))
- (untested) Bitbucket ([webhook documentation](https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push))

# Deployment requirements 
1. Redis server for task queue

2. Provide following environment variables for the container

  - WEBHOOKS_SECRET
  - REDIS_HOST
  - REDIS_PASSWORD

3. Do deployment with a `serviceAccountName` that has permissions to remove other pods and helm releases.

# Configuration

1. You have to add a custom application to your Github organisation (`https://github.com/organizations/<org name>/settings/apps/new`). Minimal configuration needs `Webhook URL` and `Webhook secret (optional)` defined. 

2. Application permissions & webhooks section, add `Repository contents` r/o permission and check `Delete (Branch or tag deleted.)` option in `Subscribe to events` section.

3. Enable this application in your organisation (`https://github.com/organizations/<org name>/settings/apps/<app name>/installations`). URL address is `<host>/webhooks` (port 80)

# Docker image build

**Automated builds:**

Tag a new release in github, docker hub integration will build and publish the images automatically.

**Manual builds (only when automated builds are not working):**

```bash
docker build --tag 'wunderio/silta-deployment-remover:latest' --tag 'wunderio/silta-deployment-remover:v1' --tag 'wunderio/silta-deployment-remover:v1.X' --tag 'wunderio/silta-deployment-remover:v1.X.Y' .
docker push wunderio/silta-deployment-remover:v1
docker push wunderio/silta-deployment-remover:v1.X
docker push wunderio/silta-deployment-remover:v1.X.Y
```
