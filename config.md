# Environment Variables

## Config

 - `LISTEN_ADDRESS` (**required**, non-empty, default: `:8080`) - The address to listen for HTTP requests on.
 - `REDIRECT_TO_LATEST` (default: `true`) - Redirect requests to `/` to the latest PDF.
 - `S3_ENDPOINT` (**required**, non-empty) - S3-compatible API endpoint.
 - `S3_REGION` - S3 region.
 - `S3_BUCKET` (**required**, non-empty) - S3 bucket name.
 - `UPLOAD_AUTH_KEY` (**required**, non-empty) - Authorization key for the `/api/upload` endpoint.
 - `UPLOAD_USER_AGENT` - User agent to use when fetching a new PDF. Will be loaded from https://github.com/jnrbsn/user-agents if empty.
 - `REAL_IP_HEADER` (default: `true`) - Get client IP address from the "Real-IP" header.
 - `LIMIT_REQUESTS` (**required**, non-empty, default: `30`) - HTTP rate limit requests.
 - `LIMIT_WINDOW` (**required**, non-empty, default: `15s`) - HTTP rate limit window.

