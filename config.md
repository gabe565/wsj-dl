# Environment Variables

## Config

 - `LISTEN_ADDRESS` (**required**, non-empty, default: `:8080`) - The address to listen for HTTP requests on.
 - `REDIRECT_TO_LATEST` (default: `true`) - Redirect requests to `/` to the latest PDF.
 - `S3_ENDPOINT` (**required**, non-empty) - S3-compatible API endpoint.
 - `S3_REGION` - S3 region.
 - `S3_BUCKET` (**required**, non-empty) - S3 bucket name.
 - `UPDATE_ON_STARTUP` (default: `true`) - Checks for an updated PDF on startup.
 - `UPDATE_CRON` (default: `0 8 * * 1-6`) - Configures the update cron interval. Leave blank to disable.
 - `UPDATE_AUTH_KEY` - Authorization key for the `/api/update` endpoint. Leave blank to disable this endpoint.
 - `UPDATE_URL` (**required**, non-empty) - URL to fetch PDFs from.
 - `UPDATE_USER_AGENT` - User agent to use when fetching a new PDF. Will be loaded from https://github.com/jnrbsn/user-agents if empty.
 - `LIMIT_REQUESTS` (**required**, non-empty, default: `5`) - HTTP rate limit requests.
 - `LIMIT_WINDOW` (**required**, non-empty, default: `10s`) - HTTP rate limit window.

