#!/bin/bash -x

# creates the necessary docker images to run testrunner.sh locally

docker build --tag="trustmachine/cppjit-testrunner" docker-cppjit
docker build --tag="trustmachine/python-testrunner" docker-python
docker build --tag="trustmachine/go-testrunner" docker-go
