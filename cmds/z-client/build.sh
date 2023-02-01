#!/bin/bash

echo "Go Build"

go build -ldflags="-s -w" -trimpath
sleep 1

echo "build electron"
npm install --save-dev @electron-forge/cli
npx electron-forge import

echo "compile to package"
npm run make