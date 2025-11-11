Package main implements the Memory Scramble game server.

Memory Scramble is a multi-player card matching game server that implements the
MIT 6.102 Problem Set 4 specification. The server provides:

  - Thread-safe concurrent gameplay for multiple players
  - HTTP API endpoints for all game operations (look, flip, replace, watch,
    restart)
  - Web-based user interface with real-time updates
  - Comprehensive game rule enforcement (Rules 1-A through 3-B)
  - Board file configuration support

Usage:

    memory-scramble -port=8080 -board=boards/perfect.txt

The server starts an HTTP server on the specified port and serves both the game
API and the web interface. Players can connect using the web interface or make
direct HTTP requests to the API endpoints.

For detailed API documentation, see the internal packages:
  - lab3/internal/board: Core game board ADT with thread safety
  - lab3/internal/commands: HTTP command interface layer
  - lab3/internal/server: HTTP server implementation with CORS
