#!/usr/bin/env python3

import socket
import sys
import os


class HTTPClient:
    def __init__(self, host, port, download_dir="downloads"):
        self.host = host
        self.port = port
        self.download_dir = download_dir

    def request(self, filename):
        try:
            response_data = self._send_request(filename)
            self._handle_response(response_data, filename)
        except ConnectionRefusedError:
            print(f"Error: Could not connect to {self.host}:{self.port}")
            sys.exit(1)
        except Exception as e:
            print(f"Error: {e}")
            sys.exit(1)

    def _send_request(self, filename):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client_socket:
            client_socket.connect((self.host, self.port))

            request = f"GET /{filename} HTTP/1.1\r\n"
            request += f"Host: {self.host}:{self.port}\r\n"
            request += "Connection: close\r\n\r\n"

            print(f"Requesting: {filename}")
            print(f"From: {self.host}:{self.port}")

            client_socket.send(request.encode("utf-8"))

            response_data = b""
            while True:
                data = client_socket.recv(4096)
                if not data:
                    break
                response_data += data

            return response_data

    def _handle_response(self, response_data, filename):
        headers, body = self._parse_response(response_data)

        if not self._is_success(headers):
            self._show_error(body)
            return

        content_type = self._get_content_type(headers)
        file_ext = os.path.splitext(filename)[1].lower()

        if self._is_html_content(content_type, file_ext):
            self._display_html(body)
        elif self._is_binary_content(content_type, file_ext):
            file_type = self._get_file_type(content_type, file_ext)
            self._save_file(body, filename, file_type)
        else:
            self._display_content(body)

    def _parse_response(self, response_data):
        header_end = response_data.find(b"\r\n\r\n")
        if header_end == -1:
            raise ValueError("Invalid HTTP response")

        headers_section = response_data[:header_end].decode("utf-8")
        body = response_data[header_end + 4 :]

        return headers_section, body

    def _is_success(self, headers):
        status_line = headers.split("\r\n")[0]
        print(f"Status: {status_line}")

        status_code = (
            status_line.split(" ")[1] if len(status_line.split(" ")) > 1 else "000"
        )
        return status_code == "200"

    def _get_content_type(self, headers):
        for line in headers.split("\r\n")[1:]:
            if line.lower().startswith("content-type:"):
                return line.split(":", 1)[1].strip().lower()
        return None

    def _is_html_content(self, content_type, file_ext):
        return (content_type and "text/html" in content_type) or file_ext in [
            ".html",
            ".htm",
        ]

    def _is_binary_content(self, content_type, file_ext):
        binary_types = ["image/png", "application/pdf"]
        binary_exts = [".png", ".pdf"]

        return (
            content_type and any(bt in content_type for bt in binary_types)
        ) or file_ext in binary_exts

    def _get_file_type(self, content_type, file_ext):
        if "image/png" in (content_type or "") or file_ext == ".png":
            return "PNG image"
        elif "application/pdf" in (content_type or "") or file_ext == ".pdf":
            return "PDF document"
        else:
            return "binary file"

    def _display_html(self, body):
        print("\n--- HTML Content ---")
        print(body.decode("utf-8", errors="ignore"))

    def _display_content(self, body):
        if self._is_text_content(body):
            print("\n--- Content ---")
            print(body.decode("utf-8", errors="ignore"))
        else:
            self._save_file(body, "unknown_file", "binary file")

    def _show_error(self, body):
        print("Error response received:")
        if body:
            print(body.decode("utf-8", errors="ignore"))

    def _save_file(self, content, filename, file_type):
        try:
            os.makedirs(self.download_dir, exist_ok=True)

            safe_filename = os.path.basename(filename) or "downloaded_file"
            file_path = os.path.join(self.download_dir, safe_filename)

            file_path = self._get_unique_filename(file_path)

            with open(file_path, "wb") as f:
                f.write(content)

            print(f"\n{file_type.capitalize()} saved to: {file_path}")
            print(f"File size: {len(content)} bytes")

        except Exception as e:
            print(f"Error saving file: {e}")

    def _get_unique_filename(self, file_path):
        if not os.path.exists(file_path):
            return file_path

        base_name, ext = os.path.splitext(file_path)
        counter = 1

        while os.path.exists(file_path):
            file_path = f"{base_name}_{counter}{ext}"
            counter += 1

        return file_path

    def _is_text_content(self, content):
        try:
            content.decode("utf-8")
            text_chars = sum(
                1 for byte in content if 32 <= byte <= 126 or byte in [9, 10, 13]
            )
            return (text_chars / len(content) > 0.7) if content else False
        except UnicodeDecodeError:
            return False


def main():
    if len(sys.argv) < 4 or len(sys.argv) > 5:
        print("Usage: uv run client.py <server_host> <server_port> <url_path> [directory]")
        print("Example: uv run client.py localhost 8080 index.html")
        print("Example: uv run client.py localhost 8080 index.html /custom/downloads")
        sys.exit(1)

    host = sys.argv[1]

    try:
        port = int(sys.argv[2])
    except ValueError:
        print("Error: Port must be a number")
        sys.exit(1)

    url_path = sys.argv[3]
    
    # Use command line directory argument, fallback to environment variable, then default
    if len(sys.argv) == 5:
        download_dir = sys.argv[4]
    else:
        download_dir = os.environ.get('DOWNLOAD_DIR', 'downloads')

    client = HTTPClient(host, port, download_dir)
    client.request(url_path)


if __name__ == "__main__":
    main()
