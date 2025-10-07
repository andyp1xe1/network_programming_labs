# Lab 1: HTTP File Server

HTTP file server using Python TCP sockets with directory listing and file download capabilities.

## Usage

```bash
# Start server
make server

# Run tests  
make test

# Download files from local server
make client FILE=index.html

# Connect to remote server on local network
make client FILE=index.html HOST=192.168.1.100
```

## Requirements Fulfilled

- HTTP file server using TCP sockets
- Serves HTML, PNG, PDF files
- 404 error handling for missing files
- Takes directory as command-line argument
- Docker Compose integration
- HTTP client implementation (2 points)
- Directory listing with hyperlinks (2 points)
- Network browsing capability (1 point)

## Lab Report

### 1. Source Directory Contents

[Screenshot: Project structure showing src/lab1/, www/, docker-compose.yml, Makefile]

The project contains server.py, client.py implementations with Docker configuration and content directory.

### 2. Docker Compose File

[Screenshot: docker-compose.yml contents]

Docker configuration defines HTTP server and client services with volume mounts and port exposure.

### 3. Starting the Container

[Screenshot: Terminal output of `make server` command]

Container starts in background mode with HTTP server service.

### 4. Server Command

[Screenshot: Server running with directory argument]

Server executes: `uv run ./packages/lab1/src/lab1/server.py ./packages/lab1/www`

### 5. Served Directory Contents

[Screenshot: Contents of www/ directory]

Directory contains index.html with embedded image, PNG file, PDF files, and subdirectory.

### 6. Browser Requests

[Screenshot: 404 error page for nonexistent.html]

404 error handling for missing files.

[Screenshot: HTML file with embedded image displayed in browser]

HTML file serving with PNG image reference.

[Screenshot: PDF file download in browser]

PDF file serving and download functionality.

[Screenshot: PNG file displayed in browser]

PNG image file serving.

### 7. Client Usage

[Screenshot: Client downloading HTML file with terminal output]

Client prints HTML content to terminal.

[Screenshot: Client downloading PDF file and saved files in downloads/]

Client saves binary files to downloads directory.

### 8. Directory Listing

[Screenshot: Generated directory listing page for subdir/]

Server generates HTML directory listing with hyperlinks for subdirectory browsing.

### 9. Network Testing

[Screenshot: Network setup showing IP configuration]

Local network setup for testing with friends' servers.

[Screenshot: Client connecting to remote server]

Successful connection to friend's server using IP address and file download.
