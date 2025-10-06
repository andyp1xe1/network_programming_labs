# Lab 1: HTTP File Server

HTTP file server using Python TCP sockets with directory listing and file download.

## Usage

```bash
# Start server
make server

# Download files from local server
make client FILE=index.html

# Connect to remote server on local network
make client FILE=index.html HOST=192.168.1.100

# Connect with custom port
make client FILE=document1.pdf HOST=192.168.1.100 PORT=9000

# Run tests
make test
```

## Features

- Serves HTML, PNG, PDF files
- Directory listing
- HTTP client for downloads
- Docker support
- Local network connectivity
