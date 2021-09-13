#!/bin/sh
docker ps | grep mls- | cut -d ' ' -f 1 | xargs docker kill
