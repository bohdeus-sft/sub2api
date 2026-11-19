#!/bin/bash
# This fork must be built from its own source, using the personal Compose file.
set -eu
cat <<'MESSAGE'
This fork uses deploy/docker-compose.personal.yml only.
Configure the required secrets and follow deploy/PERSONAL-PRIVACY.uk.md.
From the repository root, run:
  docker compose --project-directory . -f deploy/docker-compose.personal.yml up -d --build
MESSAGE
