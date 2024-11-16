#!/bin/bash

echo "!!!!!!!!!!!! STARTING CONTAINER"
docker compose up -d

echo "!!!!!!!!!!!! LOGS"
docker compose logs

echo "!!!!!!!!!!!! ENDING CONTAINER"
docker compose down
