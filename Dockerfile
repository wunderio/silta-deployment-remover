FROM node:10-alpine

RUN apk add curl bash python

# Add gcloud CLI
RUN curl -sSL https://sdk.cloud.google.com | bash \
  && rm -r /root/google-cloud-sdk/.install/.backup/
ENV PATH $PATH:/root/google-cloud-sdk/bin/

# Add kubectl
RUN yes | gcloud components install kubectl

# Install Helm
ENV HELM_VERSION v2.14.0
ENV FILENAME helm-${HELM_VERSION}-linux-amd64.tar.gz
ENV HELM_URL https://storage.googleapis.com/kubernetes-helm/${FILENAME}

RUN curl -o /tmp/$FILENAME ${HELM_URL} \
  && tar -zxvf /tmp/${FILENAME} -C /tmp \
  && rm /tmp/${FILENAME} \
  && mv /tmp/linux-amd64/helm /bin/helm \
  && helm init --client-only

# Copy node application
COPY . /app
WORKDIR "/app"

RUN npm install --production

EXPOSE 80

# Start application
ENTRYPOINT ["npm","run-script","server"]
