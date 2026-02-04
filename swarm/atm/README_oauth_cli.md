# oauth_cli (Go)

This is a small command-line tool that performs an OAuth2 installed-app loopback flow with Google and lists upcoming Calendar events.

Layout and conventions follow workspace/docs/install-how-to.md: code under ai/, binaries built to bin/, and credentials/tokens stored at workspace root.

Build and run (from workspace root):
  cd ai/swarm/atm
  go mod tidy
  go build ./cmd/oauth_cli
  ./oauth_cli

Notes:
- The program reads credentials from $WORKSPACE/credentials.json or /Users/liqiang/workspace/credentials.json by default.
- Tokens are saved to $WORKSPACE/tokens.json with 0600 permissions.
- The browser is opened using a platform-specific exec-based opener; if it fails the URL is printed.
- If port 8080 is unavailable, change the listen address and update the redirect URI in the Google Cloud Console.
