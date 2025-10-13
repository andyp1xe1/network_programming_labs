# Lab 1: HTTP File Server

HTTP file server using Python TCP sockets with directory listing and file download capabilities. The server handles one HTTP request at a time and serves HTML, PNG, and PDF files from a specified directory.

## Project Structure  

```
lab1/
├── src/lab1/
│   ├── __init__.py         # Package initialization
│   ├── server.py          # HTTP file server implementation
│   └── client.py          # HTTP client for making requests
├── www/                   # Directory containing files served by the server
│   ├── index.html         # Main HTML page with embedded image
│   └── assets/            # Static assets directory  
│       ├── style.css      # CSS styling for directory listings
│       ├── *.pdf          # PDF documents
│       ├── *.png          # PNG images
│       ├── *.jpg          # JPEG photos
│       └── *.gif          # GIF animations
├── downloads/             # Default directory for client downloads
├── Dockerfile             # Docker image for both server and client
├── docker-compose.yml     # Orchestrates server and client services
├── Makefile              # Development workflow automation
├── pyproject.toml        # Python package configuration with uv
└── .gitattributes        # Git attributes for binary assets
```

## Docker Configuration

#### Dockerfile

```dockerfile
FROM python:3.13-slim
WORKDIR /workspace
COPY pyproject.toml uv.lock ./
RUN pip install uv && uv sync
COPY . .
EXPOSE 8080
```

#### Docker Compose

```yaml
services:
  http-server:
    build:
      context: ../..
      dockerfile: packages/lab1/Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - ./www:/workspace/packages/lab1/www
    command: ["uv", "run", "./packages/lab1/src/lab1/server.py", "/workspace/packages/lab1/www", "8080"]
    
  http-client:
    build:
      context: ../..
      dockerfile: packages/lab1/Dockerfile
    profiles:
      - client
    volumes:
      - ${DOWNLOAD_DIR:-./downloads}:/workspace/downloads
    command: ["uv", "run", "./packages/lab1/src/lab1/client.py", "${SERVER_HOST:-http-server}", "${SERVER_PORT:-8080}", "${URL_PATH:-index.html}", "/workspace/downloads"]
    depends_on:
      - http-server
```

## Starting the Container

![Starting Container](./img/make_dev.png)

Container starts in watch mode with HTTP server service.

```bash
# Start development environment with file watching
make dev

# Start server only
make server
```

## Server Usage

The server takes a directory to serve as a command-line argument and handles one HTTP request at a time.

```bash
# Docker execution
make server           # Serves www/ directory on port 8080

# Direct execution with uv
uv run server www/ 8080

# Local Python execution  
python src/lab1/server.py www/ 8080
```

**Server Features:**
- **Single-threaded HTTP Server** - Handles one request at a time using Python's socket module
- **File Type Support** - Serves HTML, PNG, and PDF files with proper MIME types
- **Directory Listing** - Generates HTML pages for directory browsing with navigation
- **Error Handling** - Returns HTTP 404 for non-existent files or unknown extensions
- **Path Security** - Prevents directory traversal attacks
- **Custom Styling** - Styled directory listings and error pages

## Served Directory Contents

```
www/
├── index.html         # HTML file with embedded PNG image reference
└── assets/            # Subdirectory with various files
    ├── style.css      # Styling for directory listings
    ├── nujabes.png    # PNG image (referenced in HTML)
    ├── samurai_champloo.gif
    ├── 1962-calhoun.pdf
    ├── descrierea-moldovei.pdf
    ├── zipclasspaper.pdf
    └── photo_*.jpg    # Multiple JPEG photos
```

## Browser Requests

#### 404 Error for Non-existent File
![404 Error](./img/nonexistent_web.png)

#### HTML File with Embedded Image  
![HTML with Image](./img/embeded_web.jpg)

The HTML file references a PNG image using `<img>` tag, demonstrating proper file serving.

#### PDF File Serving
![PDF File](./img/pdf_web.png)

PDF files are served with correct content-type for browser display/download.

#### PNG Image Display
![PNG Image](./img/image_web.png)

PNG images are served with proper image/png MIME type.

## Directory Listing

#### Root Directory Listing
![Root Directory](./img/web_listing_root.png)

Generated HTML page showing directory contents with hyperlinks.

#### Subdirectory Navigation
![Subdirectory](./img/web_listing_subdir.png)

Directory listing for assets/ subdirectory with parent directory (`../`) navigation.

## Client Implementation

The HTTP client takes command-line arguments and handles different file types appropriately.

```bash
# Client usage format:
# client.py server_host server_port url_path directory

# Docker execution
make client FILE=index.html                    # Display HTML content
make client FILE=assets/nujabes.png           # Download PNG image  
make client FILE=assets/                      # Show directory listing

# Direct execution
uv run client localhost 8080 index.html downloads/
uv run client localhost 8080 assets/1962-calhoun.pdf downloads/
```

**Client Behavior:**
- **HTML/Directory Listings** - Prints response body as-is to terminal
- **PNG/PDF Files** - Saves files to specified directory
- **Error Handling** - Displays server error responses
- **File Management** - Creates download directory and handles filename conflicts

## Development Workflow

#### Makefile Commands

```bash
make dev        # Start development with file watching
make server     # Start server in background  
make client     # Run client with configurable parameters
make test       # Run automated test suite
make logs       # View server logs
make status     # Check service status
make stop       # Stop all services
make clean      # Clean up containers and downloads
```

#### Automated Testing

The test suite (`make test`) verifies:
1. HTML file download and display
2. PDF file download and saving
3. Directory browsing functionality  
4. 404 error handling for non-existent files

## Implementation Details

#### Server (HTTPServer Class)
- Single-threaded socket-based HTTP/1.1 server
- Parses GET requests and serves files from specified directory
- Generates directory listings with HTML and CSS styling
- Returns appropriate HTTP status codes (200, 404, 403, 405, 500)
- Prevents directory traversal with path validation

#### Client (HTTPClient Class)  
- Raw socket HTTP client with response parsing
- Content-type aware file handling (HTML display vs binary download)
- Automatic download directory creation
- Connection error handling and cleanup

---

*Built with Python 3.13, TCP sockets, uv package manager, and Docker containerization*
