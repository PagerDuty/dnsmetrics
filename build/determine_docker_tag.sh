#!/bin/bash

if [ -n "$TRAVIS" ]; then
  # Only build on pushes / merges to master branch; don't build pull requests
  if [ "$TRAVIS_PULL_REQUEST" = "false" -a "$TRAVIS_BRANCH" = "master" ]; then
    echo -n $TRAVIS_COMMIT
  fi
else
  echo -n `git rev-parse HEAD`
fi
