#!/usr/bin/env bash

# gp run ./build.sh
gox -osarch="linux/amd64" -output="./bin/convergence" github.com/IAD-ZHDK/Convergence
