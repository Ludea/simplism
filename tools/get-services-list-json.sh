#!/bin/bash
set -o allexport; source .simplismconfig.env; set +o allexport

curl ${DISCOVERY_SERVICE_URL} \
-H 'content-type:application/json; charset=UTF-8' \
-H "admin-discovery-token:${ADMIN_DISCOVERY_TOKEN}"
