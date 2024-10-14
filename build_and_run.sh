#!/bin/bash

# Navigate to the Go project directory
cd ../WSA-Vision/electron-app/

# Build the Go executable
go build -o go_build_WSA_WSA

# Navigate back to the Electron app directory
cd ../electron-app

# Copy the Go executable to the Electron app directory
cp ../WSA-Vision/go_build_WSA_WSA* ./

# Install dependencies (if not already installed)
npm install

# Start the Electron app
npm start
