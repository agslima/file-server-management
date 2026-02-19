#!/usr/bin/env bash
set -euo pipefail

composer install --no-interaction
./vendor/bin/phpunit -c phpunit.xml
