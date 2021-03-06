FROM node:15.11.0-alpine3.10
ADD package.json .
RUN npm install --legacy-peer-deps --ignore-scripts
ADD mm_client.mjs .