#!/usr/bin/env bash

# gp run iadzhdk "./build.sh"
gox -osarch="linux/amd64" -output="./bin/convergence" github.com/IAD-ZHDK/Convergence
