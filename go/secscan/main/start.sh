#!/bin/sh
set -e
# saichler/secscan-postgres:latest's own /start-postgres.sh exists but its
# ENTRYPOINT is commented out in the base image (verified: same gap found
# in ../l8alarms's real alm/main/Dockerfile, which also never starts
# postgres) -- starting it here, before execing our own binary, is what
# actually makes the StatefulSet's embedded-Postgres design (PRD §16, own
# base image saichler/secscan-postgres) work.
/start-postgres.sh secscan secscan secscan 5432
exec /home/run/secscan
