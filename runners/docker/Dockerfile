FROM node:15.11.0-alpine3.10
ADD . /app
WORKDIR /app
RUN ./setup.sh
RUN npm install --legacy-peer-deps --ignore-scripts
ADD node_modules/matterless node_modules/matterless
EXPOSE 8080 8081

ENTRYPOINT ["mls-lambda"]