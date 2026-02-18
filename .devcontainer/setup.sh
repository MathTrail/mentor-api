#!/bin/bash
set -e
echo "MathTrail: Bootstrapping from infra-local..."
curl -sSL https://raw.githubusercontent.com/MathTrail/infra-local/main/scripts/bootstrap.sh | bash
