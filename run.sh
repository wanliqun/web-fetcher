#!/bin/bash

# check if there are any arguments
if [ $# -eq 0 ]; then
  echo "Usage: ./run.sh [--build | --metadata] [urls]"
  echo "Example: ./run.sh --build --metadata https://www.google.com https://www.bing.com"
  echo "The --build flag (optional) builds the Docker image (first time only)"
  echo "The --metadata flag (optional) enables printing metadata about fetched web page"
  exit 1
fi

# Extract arguments from the command line
args=()
metaArg=""
buildDockerImage=false

for arg in "$@"; do
  if [[ $arg == "--metadata" || $arg == "-a" ]]; then
    metaArg="$arg"
  elif [[ $arg == "--build" ]]; then
    buildDockerImage=true
  else
    # Collect URLs as regular arguments
    if [[ $arg =~ ^https?:// ]]; then
      args+=("$arg")
    fi
  fi
done

# check if there are any urls after the optional flag
if [ -z "$args" ]; then
  echo "No urls provided"
  exit 1
fi

if $buildDockerImage;  then
    # Build the Docker image
    docker build -t web-fetcher .

    # check if the build was successful
    if [ $? -ne 0 ]; then
        echo "Failed to build the docker image"
        exit 1
    fi
fi

# Run the Docker container with the specified arguments
docker run --rm -v ${PWD}:/app/output web-fetcher ${metaArg} "${args[@]}"