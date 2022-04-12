FROM node:10-alpine

RUN apk add curl bash python jq

# Add gcloud CLI
RUN curl -sSL https://sdk.cloud.google.com | bash \
  && rm -r /root/google-cloud-sdk/.install/.backup/
ENV PATH $PATH:/root/google-cloud-sdk/bin/

# Add kubectl
RUN yes | gcloud components install kubectl

# Install Helm
ENV HELM_VERSION v3.0.2

RUN curl -o /tmp/helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz \
  && tar -zxvf /tmp/helm.tar.gz -C /tmp \
  && rm /tmp/helm.tar.gz \
  && find /tmp \
  && mv /tmp/linux-amd64/helm /bin/helm

# Copy node application
COPY /app /app
WORKDIR "/app"

RUN npm install --production

EXPOSE 80

# Start application
ENTRYPOINT ["npm","run-script","server"]
