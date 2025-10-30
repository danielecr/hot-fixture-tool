# Hot Fixture Tool Daemon (hfitd)

A secure REST API service for the Hot Fixture Tool that provides database and file access with public key authentication.

## Features

- **Public Key Authentication**: Uses RSA public/private key cryptography for secure client authentication
- **JWT Token-based Authorization**: Issues JWT tokens for authenticated sessions
- **Database Access**: REST endpoints for database inspect and export operations
- **File Management**: Endpoints for file listing and downloading
- **Secure by Default**: All API endpoints are protected except authentication routes

## Authentication Flow

The authentication system uses a challenge-response mechanism with public key cryptography:

1. **Request Challenge** (`POST /auth/challenge`)
   - Client sends username
   - Server responds with a random challenge and expiration time

2. **Authenticate** (`POST /auth/authenticate`)
   - Client signs the challenge with their private key
   - Server verifies the signature using the configured public key
   - Server issues a JWT token on successful verification

3. **Access Protected Endpoints**
   - Client includes JWT token in Authorization header: `Bearer <token>`
   - Server validates token for all protected routes

## API Endpoints

### Authentication (Unprotected)
- `POST /auth/challenge` - Request authentication challenge
- `POST /auth/authenticate` - Submit signed challenge for JWT token
- `GET /health` - Health check endpoint

### DBMS Providers Operations (Protected)
- `GET /db/dbmss` - List available providers (access)

### Database Operations (Protected)
- `GET /db/{dbms}/dbs` - List available databases
- `GET /db/{dbms}/{dbid}/tables` - List tables in a database
- `GET /db/{dbms}/{dbid}/table/{tableid}/rows` - List rows in a table
- `GET /db/{dbms}/{dbid}/table/{tableid}/rows?filter="where keyfield>3 and keyfield<100"` - List rows in a table


### Volume Operations (Protected)
- `GET /volumes` - List available volumes

### File Operations (Protected)
- `GET /files/{volume}/list` - **Stream** available files (NDJSON format, optimized for large directories)
- `GET /files/{volume}/list?filter[]=name:*.log&filter[]=mtime:-7&filter[]=size:>1024` - Stream filtered files
- `GET /files/{volume}/download?path=<filepath>` - Download a file
- `GET /files/{volume}/download?folder=<folder>&filter[]=name:*.config&filter[]=sort:mtime:desc` - Download best matching file

#### File Download with Smart Selection

The download endpoint supports intelligent file selection when downloading from folders with multiple matching files:

**Smart Download Examples:**
```bash
# Download newest log file
GET /files/logs/download?folder=app&filter[]=name:*.log&filter[]=sort:mtime:desc

# Download largest backup file  
GET /files/backups/download?folder=daily&filter[]=name:backup_*&filter[]=sort:size:desc

# Download alphabetically first config file
GET /files/config/download?folder=env&filter[]=name:*.conf&filter[]=sort:name:asc
```

**Sorting Options for Download:**
- `filter[]=sort:mtime:desc` - Newest file first (most recent modification)
- `filter[]=sort:mtime:asc` - Oldest file first
- `filter[]=sort:size:desc` - Largest file first
- `filter[]=sort:size:asc` - Smallest file first  
- `filter[]=sort:name:desc` - Alphabetically last (Z-A)
- `filter[]=sort:name:asc` - Alphabetically first (A-Z)
- `filter[]=sort:path:asc` - By full path

**Performance Optimization:**
- **O(n) algorithm**: Single-pass file selection without full sorting
- **Streaming filters**: Files filtered during directory traversal
- **Memory efficient**: Only best candidate kept in memory

#### File Listing Performance & Streaming

The `/files/{volume}/list` endpoint uses **streaming NDJSON** (`application/x-json-stream`) for optimal performance with large directories, similar to Unix `find` command:

**Streaming Response Format:**
```
Content-Type: application/x-json-stream

{"name":"file1.txt","path":"dir/file1.txt","size":1024,"modtime":1735574400,"isdir":false}
{"name":"file2.log","path":"logs/file2.log","size":2048,"modtime":1735574401,"isdir":false}
{"name":"config.yml","path":"config.yml","size":512,"modtime":1735574402,"isdir":false}
```

**Advanced Filtering:**
- `filter[]=name:*.log` - Glob pattern matching (* and ? wildcards)
- `filter[]=mtime:-7` - Files modified before 7 days ago (negative = before)
- `filter[]=mtime:7` - Files modified within last 7 days (positive = after)
- `filter[]=size:>1024` - Files larger than 1024 bytes
- `filter[]=size:<1048576` - Files smaller than 1MB
- `filter[]=size:2048` - Files exactly 2048 bytes

**Performance Benefits:**
- **Real-time streaming**: Files appear immediately as found
- **Memory efficient**: No buffering of entire directory listing
- **Unix find performance**: Optimized for millions of files
- **Early filtering**: Filters applied during traversal, not after

### Pack Download exec Operation (Protected)
- `POST /packdownload/{packname}` accept yaml payload to create a package.tar.gz with all files described in the yaml, then it returns the file to the caller.

Package request creation is logged into redis with key schema: <useremail>_pkg_<packname>

and content the yaml definition.
Another entry in redis has the key schema: <useremail>_pkg_<packname>_timestamp
and content {"timestamp": <timestamp>, "size": <pkgsize>}

## Configuration

Configure the service using environment variables. You can use a `.env` file for convenience:

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` with your configuration:**
   ```bash
   # Edit the values to match your setup
   nano .env
   ```

3. **Or set environment variables directly:**

### Server Configuration
- `SERVER_ADDRESS` - Server bind address (e.g., `:8080`)

### Database Configuration
- `DB_HOST` - Database host
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name

### Authentication Configuration
- `JWT_SECRET` - Secret key for signing JWT tokens (keep secure!)

### Redis Configuration
- `REDIS_URL` - Redis connection URL (e.g., `redis://localhost:6379`)

### Admin Configuration
- `HFITD_SOCKET_PATH` - Unix socket path for admin CLI (default: `/tmp/hfitd.sock`)

## Administration

The system includes a built-in CLI tool `hfitd-cli` for administration:

### CLI Commands
```bash
# Add a user with their public key
hfitd-cli adduser <email> <public_key_file_or_content>

# Renew JWT signing keys (invalidates existing tokens)
hfitd-cli renew-jwt

# Get current JWT public key for verification
hfitd-cli get-jwt-public-key
```

### Admin Socket
- The server creates a Unix socket for secure local administration
- Only the server owner can access the socket (permissions: 600)
- No network exposure for admin operations

## API Endpoints

### Well-Known Endpoints
- `GET /.well-known/jwks.json` - JWT public key in JWKS format (RFC 7517)

### Authentication (Unprotected)
- `POST /auth/challenge` - Request authentication challenge
- `POST /auth/authenticate` - Submit signed challenge for JWT token
- `GET /health` - Health check endpoint
The system stores **per-user public keys** in Redis:

- **Key Pattern**: `user__<user_email>` 
- **Value**: RSA public key in PEM format
- **Purpose**: Each user has their own public key for authentication
- **Security Model**: 
  - Each user generates their own RSA key pair
  - User's public key is stored in Redis (server-side)
  - User keeps their private key (client-side)
  - User signs challenges with their private key
  - Server verifies with user's specific public key from Redis

**Example Redis entries:**
```
user__alice@example.com -> "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhki..."
user__bob@company.com   -> "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhki..."
```

**Benefits:**
- ✅ Multi-user support with individual authentication
- ✅ Scalable user management
- ✅ Easy user onboarding/offboarding
- ✅ No shared authentication secrets

**Note:** The `.env` file is automatically loaded if present. System environment variables take precedence over `.env` file values.

## Example Usage

### 1. Setup Configuration
```bash
# Copy and edit the environment file
cp .env.example .env
# Edit .env with your actual values
```

### 2. Setup Redis and Add Users
```bash
# Start Redis (using Docker)
docker run -d -p 6379:6379 redis:alpine

# Or install Redis locally and start it
redis-server
```

### 3. Generate Key Pair (for each user)
```bash
# Use the provided script (recommended)
./generate_keys.sh

# Or manually with OpenSSL:
# Generate private key
openssl genrsa -out alice_private_key.pem 2048

# Extract public key
openssl rsa -in alice_private_key.pem -pubout -out alice_public_key.pem
```

### 4. Add User Public Keys via CLI
```bash
# Generate key pair for a user
./generate_keys.sh

# Rename keys for the user
mv private_key.pem alice_private_key.pem  
mv public_key.pem alice_public_key.pem

# Add user to the system (server must be running)
./hfitd-cli adduser alice@example.com alice_public_key.pem

# Or add user with inline public key
./hfitd-cli adduser bob@company.com "-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----"

# Verify user was added (check server logs)
```

### 5. JWT Key Management
```bash
# Renew JWT signing keys (invalidates existing tokens)
./hfitd-cli renew-jwt

# Get current JWT public key (for token verification)
./hfitd-cli get-jwt-public-key
```

**JWT Public Key Endpoint:**
The server exposes the JWT public key at: `GET /.well-known/jwks.json`

### 5. Key Distribution
- **Server**: Accesses user public keys from Redis
- **Each User**: Keeps their own private key secure
- **Security**: Private key files should have restricted permissions (`chmod 600 private_key.pem`)

### 6. Set Environment Variables (alternative to .env)
```bash
export SERVER_ADDRESS=":8080"
export DB_HOST="localhost"
export DB_USER="myuser"
export DB_PASSWORD="mypassword"
export DB_NAME="mydatabase"
export JWT_SECRET="your-secret-key-here"
export PUBLIC_KEY_PEM="$(cat public_key.pem)"
```

### 3. Run the Service
```bash
go run main.go
```

### 4. Client Authentication Example (Go)
```go
// Request challenge
challengeReq := ChallengeRequest{Username: "user"}
// ... make HTTP POST to /auth/challenge

// Sign challenge with private key
signature := signChallenge(challenge, privateKey)

// Authenticate
authReq := AuthRequest{
    Username:  "user",
    Challenge: challenge,
    Signature: base64.StdEncoding.EncodeToString(signature),
}
// ... make HTTP POST to /auth/authenticate

// Use returned JWT token in subsequent requests
headers := map[string]string{
    "Authorization": "Bearer " + token,
}
```

## Security Considerations

1. **Private Key Security**: Client private keys must be kept secure and never transmitted
2. **JWT Secret**: Use a strong, randomly generated JWT secret
3. **HTTPS**: Always use HTTPS in production to protect tokens in transit
4. **Token Expiration**: JWT tokens expire after 24 hours by default
5. **Challenge Expiration**: Authentication challenges expire after 5 minutes

## Building and Running

```bash
# Install dependencies
go mod tidy

# Build the application
go build

# Run the service
./hfitd
```

## Dependencies

- [gorilla/mux](https://github.com/gorilla/mux) - HTTP router
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) - JWT token handling
- [lib/pq](https://github.com/lib/pq) - PostgreSQL driver