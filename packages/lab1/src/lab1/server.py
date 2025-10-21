#!/usr/bin/env python3

import socket
import sys
import os
import urllib.parse
import mimetypes
import time
import argparse


class HTTPServer:
    def __init__(self, directory, port, simulate_delay=False, delay_seconds=0.1):
        self.directory = os.path.abspath(directory)
        self.port = port
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.simulate_delay = simulate_delay
        self.delay_seconds = delay_seconds

    def start(self):
        self.socket.bind(("0.0.0.0", self.port))
        self.socket.listen(5)
        print(f"HTTP Server started on port {self.port}")
        print(f"Serving directory: {self.directory}")
        print(f"Simulate delay: {self.simulate_delay} ({self.delay_seconds}s)")

        try:
            while True:
                client_socket, address = self.socket.accept()
                print(f"Connection from {address}")
                self.handle_request(client_socket)
                client_socket.close()
        except KeyboardInterrupt:
            print("\nServer shutting down...")
        finally:
            self.socket.close()

    def handle_request(self, client_socket):
        try:
            # Simulate work delay if enabled
            if self.simulate_delay:
                time.sleep(self.delay_seconds)
                
            request_data = client_socket.recv(4096).decode("utf-8")
            if not request_data:
                return

            print(f"Request: {request_data.split()[0:3]}")

            request_line = request_data.split("\n")[0].strip()
            if not request_line:
                return

            parts = request_line.split()
            if len(parts) < 2:
                self.write_response(client_socket, 400, "Bad Request")
                return

            method, path = parts[0], parts[1]

            if method != "GET":
                self.write_response(client_socket, 405, "Method Not Allowed")
                return

            self.serve_file(client_socket, path)

        except Exception as e:
            print(f"Error handling request: {e}")
            self.write_response(client_socket, 500, "Internal Server Error")

    def serve_file(self, client_socket, path):
        path = urllib.parse.unquote(path)
        if path.startswith("/"):
            path = path[1:]

        full_path = os.path.abspath(os.path.join(self.directory, path))

        if not full_path.startswith(self.directory):
            self.write_response(client_socket, 403, "Forbidden")
            return

        if os.path.isdir(full_path):
            self.serve_directory(client_socket, full_path, path)
            return

        if not os.path.exists(full_path) or not os.path.isfile(full_path):
            self.write_response(client_socket, 404, "Not Found")
            return

        file_ext = os.path.splitext(full_path)[1].lower()
        if file_ext not in [".html", ".htm", ".png", ".pdf", ".css", ".gif", ".jpg", ".jpeg"]:
            self.write_response(client_socket, 404, "Not Found")
            return

        try:
            with open(full_path, "rb") as file:
                content = file.read()

            content_type = self.get_content_type(full_path, file_ext)
            self.write_response(
                client_socket, 200, "OK", content, {"Content-Type": content_type}
            )

        except Exception as e:
            print(f"Error serving file: {e}")
            self.write_response(client_socket, 500, "Internal Server Error")

    def serve_directory(self, client_socket, full_path, relative_path):
        try:
            files = sorted(os.listdir(full_path))
            html_content = self.generate_directory_listing(
                relative_path, files, full_path
            )
            self.write_response(
                client_socket,
                200,
                "OK",
                html_content.encode("utf-8"),
                {"Content-Type": "text/html"},
            )

        except Exception as e:
            print(f"Error serving directory: {e}")
            self.write_response(client_socket, 500, "Internal Server Error")

    def write_response(
        self, client_socket, status_code, status_text, content=None, headers=None
    ):
        if content is None:
            content = self.generate_error_page(status_code, status_text).encode("utf-8")
        if headers is None:
            headers = {"Content-Type": "text/html"}

        response = f"HTTP/1.1 {status_code} {status_text}\r\n"

        for key, value in headers.items():
            response += f"{key}: {value}\r\n"

        response += f"Content-Length: {len(content)}\r\n"
        response += "Connection: close\r\n"
        response += "\r\n"

        client_socket.send(response.encode("utf-8"))
        client_socket.send(content)

    def get_content_type(self, full_path, file_ext):
        content_type = mimetypes.guess_type(full_path)[0]
        if content_type is None:
            content_type_map = {
                ".html": "text/html",
                ".htm": "text/html",
                ".png": "image/png",
                ".pdf": "application/pdf",
                ".css": "text/css",
                ".gif": "image/gif",
                ".jpg": "image/jpeg",
                ".jpeg": "image/jpeg",
            }
            content_type = content_type_map.get(file_ext, "application/octet-stream")
        return content_type

    def generate_error_page(self, code, message):
        return f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{code} {message}</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="error-container">
        <div class="error-logo">moss is sentient btw</div>
        <h1>{code} {message}</h1>
        <p>The requested resource could not be found or accessed.</p>
    </div>
</body>
</html>"""

    def generate_directory_listing(self, relative_path, files, full_path):
        html_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Directory listing for /{relative_path}</title>
    <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">moss is sentient btw</div>
            <h1>/{relative_path}</h1>
        </div>
        <div class="listing">
            <ul>
"""

        if relative_path:
            parent_path = (
                "/".join(relative_path.split("/")[:-1]) if "/" in relative_path else ""
            )
            html_content += f'<li><a href="/{parent_path}" class="dir">../</a></li>\n'

        for filename in files:
            file_path = os.path.join(full_path, filename)
            url_path = f"/{relative_path}/{filename}".replace("//", "/")

            if os.path.isdir(file_path):
                html_content += (
                    f'<li><a href="{url_path}" class="dir">{filename}/</a></li>\n'
                )
            else:
                html_content += f'<li><a href="{url_path}">{filename}</a></li>\n'

        html_content += "</ul></div></div></body></html>"
        return html_content


def main():
    # Read environment variables with defaults
    env_delay = float(os.getenv('HTTP_DELAY', '0.1'))
    env_no_delay = os.getenv('HTTP_NO_DELAY', 'false').lower() == 'true'
    env_port = int(os.getenv('HTTP_PORT', '8080'))
    env_directory = os.getenv('HTTP_DIRECTORY', './www')
    
    parser = argparse.ArgumentParser(
        description="Single-threaded HTTP File Server",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s ./www 8080
  %(prog)s ./www 8080 --delay 0.5
  %(prog)s ./www 8080 --no-delay

Environment Variables:
  HTTP_DIRECTORY      Directory to serve (default: ./www)
  HTTP_PORT          Port number (default: 8080)
  HTTP_DELAY         Delay in seconds (default: 0.1)
  HTTP_NO_DELAY      Disable delay (default: false)
        """
    )
    
    parser.add_argument("directory", nargs='?', default=env_directory, help=f"Directory to serve files from (default: {env_directory})")
    parser.add_argument("port", nargs='?', type=int, default=env_port, help=f"Port number to listen on (default: {env_port})")
    
    # Server behavior options
    parser.add_argument("--delay", type=float, default=env_delay, metavar="SECONDS",
                        help=f"Simulation delay per request in seconds (default: {env_delay})")
    parser.add_argument("--no-delay", action="store_true", default=env_no_delay,
                        help="Disable simulation delay")
    
    args = parser.parse_args()
    
    # Validate directory
    if not os.path.exists(args.directory):
        print(f"Error: Directory '{args.directory}' does not exist")
        sys.exit(1)

    if not os.path.isdir(args.directory):
        print(f"Error: '{args.directory}' is not a directory")
        sys.exit(1)
    
    # Configure options
    simulate_delay = not args.no_delay
    
    # Show configuration
    print("=== Server Configuration ===")
    print(f"Directory: {args.directory}")
    print(f"Port: {args.port}")
    print(f"Delay: {args.delay}s (enabled: {simulate_delay})")

    server = HTTPServer(
        directory=args.directory,
        port=args.port,
        simulate_delay=simulate_delay,
        delay_seconds=args.delay
    )
    server.start()


if __name__ == "__main__":
    main()
