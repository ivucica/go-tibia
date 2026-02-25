#!/bin/bash
# e.g. sha256:c8925f8de1b6f168eda664b6ed14ff14e59e063466209e7f335dd0f5b1746f7f
docker run --rm  -v "$(realpath .)":/workspace -it sourcegraph/scip-go:latest bash -c 'cd /workspace && scip-go'

# via: https://github.com/sourcegraph/jsonrpc2/blob/3c4c92ad61e8a64c37816d2c573f5d0094d96d33/.github/workflows/scip.yml

