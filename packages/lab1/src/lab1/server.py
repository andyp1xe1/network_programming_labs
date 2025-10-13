#!/usr/bin/env python3

import socket
import sys
import os
import urllib.parse
import mimetypes


class HTTPServer:
    def __init__(self, directory, port):
        self.directory = os.path.abspath(directory)
        self.port = port
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

    def start(self):
        self.socket.bind(("0.0.0.0", self.port))
        self.socket.listen(5)
        print(f"HTTP Server started on port {self.port}")
        print(f"Serving directory: {self.directory}")

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
    if len(sys.argv) != 3:
        print("Usage: uv run server.py <directory> <port>")
        sys.exit(1)

    directory = sys.argv[1]
    try:
        port = int(sys.argv[2])
    except ValueError:
        print("Error: Port must be a number")
        sys.exit(1)

    if not os.path.exists(directory):
        print(f"Error: Directory '{directory}' does not exist")
        sys.exit(1)

    if not os.path.isdir(directory):
        print(f"Error: '{directory}' is not a directory")
        sys.exit(1)

    server = HTTPServer(directory, port)
    server.start()


if __name__ == "__main__":
    main()
