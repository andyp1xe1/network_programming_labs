# Lab 1: HTTP File Server with TCP Sockets

This project implements a simple HTTP file server using Python TCP sockets,
similar to Python's `http.server`. It supports serving HTML, PNG, and PDF files
with directory listing functionality.

## Features

- HTTP file server with TCP sockets
- Supports HTML, PNG, and PDF file types
- Directory listing with nested directory support
- HTTP client for downloading files
- 404 error handling for missing files
- Docker Compose setup for easy deployment
- UV package manager integration

## Project Structure

```
lab1/
├── server.py          
├── client.py          
├── docker-compose.yml 
├── Dockerfile         
├── pyproject.toml     
├── uv.lock           
├── Makefile          
├── www/              
│   ├── index.html    
│   ├── sample.png    
│   ├── document1.pdf 
│   ├── document2.pdf 
│   └── subdir/       
└── downloads/        
```

## Usage

1. **Start the HTTP server:**
   ```bash
   make server
   ```

2. **Run the HTTP client:**
   ```bash
   make client FILE=index.html
   ```

3. **Other commands:**
   ```bash
   make test      # Run automated tests
   make build     # Build Docker images
   make clean     # Clean up containers and downloads
   make downloads # View downloaded files
   ```

## HTTP Server Features

### Supported File Types
- **HTML** (.html, .htm): Served with `text/html` content type
- **PNG** (.png): Served with `image/png` content type
- **PDF** (.pdf): Served with `application/pdf` content type

### Directory Listing
- Automatically generates HTML directory listings
- Shows files and subdirectories with proper links
- Includes parent directory navigation (`../`)
- Styled with CSS for better appearance

### Error Handling
- **404 Not Found**: For missing files or unsupported file types
- **403 Forbidden**: For access outside server directory
- **405 Method Not Allowed**: For non-GET requests
- **500 Internal Server Error**: For server-side errors

## HTTP Client Features

- **HTML files**: Prints content to console
- **PNG/PDF files**: Downloads to specified directory
- **Directory listings**: Prints HTML content to console

## Implementation Details

### Server Architecture
- Single-threaded server handling one request at a time
- TCP socket listening on specified port
- HTTP/1.1 compliant responses
- Path traversal protection
- URL decoding for proper file access

### Client Architecture
- TCP socket connection to server
- HTTP/1.1 request generation
- Response parsing with header/body separation
- Automatic file type detection
- Binary/text content handling

## Security Features

- Path traversal protection (prevents `../` attacks)
- File type validation (only serves allowed extensions)
- Input validation for command line arguments
- Proper error handling without information disclosure

---

**Course**: Networks Programming  
**Lab**: 1 - HTTP File Server with TCP Sockets  
**Requirements**: Docker Compose, UV, Python 3.13+
