# hfit CLI cmd to interact with remote hfitd

This is a command line tool that connects to a remote hfitd server using public key authentication.

It stores configuration in `$HOME/.hfit/config` containing:
- HFITD_HOST
- EMAIL 
- PUBLIC_KEY (path to private key file for signing)

## Installation

```bash
cd hfit/
go build
```

## Configuration

First, configure the CLI with your server details and authentication:

```bash
./hfit config https://your-hfitd-server.com user@example.com /path/to/your/private_key.pem
```

## Authentication

Before making API calls, authenticate to get a JWT token:

```bash
./hfit login
```

This performs a public key challenge-response authentication and stores the JWT token in `~/.hfit/token`.

## Commands

The CLI tool contacts the remote hfitd server and provides the following commands:

### Database Operations
```bash
# List all databases
./hfit dbs

# List tables in a specific database
./hfit tables <db_id>

# List rows in a specific table
./hfit rows <db_id> <table_id>
```

### File Operations
```bash
# List files
./hfit files

# Download a file
./hfit download <file_path>
```

## Example Workflow

```bash
# 1. Configure the CLI
./hfit config https://localhost:8080 alice@example.com ~/.ssh/id_rsa

# 2. Authenticate and get JWT token
./hfit login

# 3. List available databases
./hfit dbs

# 4. List tables in database "db1"
./hfit tables db1

# 5. Get rows from table "users" in database "db1"
./hfit rows db1 users

# 6. List available files
./hfit files

# 7. Download a specific file
./hfit download /path/to/file.txt
```

## Authentication Flow

1. The CLI reads your email and private key from the configuration
2. It requests a challenge from the server using your email
3. The CLI signs the challenge with your private key
4. The server verifies the signature using your stored public key
5. If valid, the server returns a JWT token
6. The JWT token is stored locally and used for subsequent API calls
