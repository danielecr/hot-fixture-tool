# Hot Fixture Tool Daemon (hfitd)

A secure REST API service for the Hot Fixture Tool that provides database and file access with public key authentication.

## Features

- **Public Key Authentication**: Uses RSA public/private key cryptography for secure client authentication
- **JWT Token-based Authorization**: Issues JWT tokens for authenticated sessions
- **Database Access**: REST endpoints for database operations
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

### Database Operations (Protected)
- `GET /db/dbs` - List available databases
- `GET /db/{dbid}/tables` - List tables in a database
- `GET /db/{dbid}/table/{tableid}/rows` - List rows in a table

### File Operations (Protected)
- `GET /files/list` - List available files
- `GET /files/download?path=<filepath>` - Download a file

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