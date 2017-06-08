#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
entrustdir="$workspace/src/github.com/trust-tech"
if [ ! -L "$entrustdir/go-trustmachine" ]; then
    mkdir -p "$entrustdir"
    cd "$entrustdir"
    ln -s ../../../../../. go-trustmachine
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$entrustdir/go-trustmachine"
PWD="$entrustdir/go-trustmachine"

# Launch the arguments with the configured environment.
exec "$@"
