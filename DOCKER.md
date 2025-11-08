# Docker Deployment Guide

This guide shows how to deploy Hot Fixture Tool using Docker and Docker Compose.

## Quick Start

### 1. Deploy the Stack
```bash
# Clone the repository
git clone https://github.com/danielecr/hot-fixture-tool.git
cd hot-fixture-tool

# Start the services (Redis + hfitd)
docker-compose up -d

# Check service status
docker-compose ps
```

### 2. Initialize Admin Access
```bash

# Add user via CLI
docker-compose exec hfitd ./hfitd-cli adduser devuser@yourdomain.com "$(cat public_key.pem)"
```

### 3. Test Authentication
```bash
# Request challenge
hfit login

# Get JWT public key for verification
curl http://localhost:8080/.well-known/jwks.json
```

## Administration Commands

### User Management
```bash
# Add a new user
docker-compose exec hfitd ./hfitd-cli adduser alice@company.com "$(cat alice_public_key.pem)"

# Add user with inline public key
docker-compose exec hfitd ./hfitd-cli adduser bob@company.com "-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----"
```

### JWT Key Management
```bash
# Renew JWT signing keys (invalidates existing tokens)
docker-compose exec hfitd ./hfitd-cli renew-jwt

# Get current JWT public key
docker-compose exec hfitd ./hfitd-cli get-jwt-public-key
```

### Monitoring
```bash
# View logs
docker-compose logs -f hfitd

# Check health
curl http://localhost:8080/health

# Monitor all services
docker-compose logs -f
```

## File Operations
```bash
# Copy files to the data directory for access via API
cp /path/to/your/files/* ./hotfixtool_data/

# The files will be available via the API at /files/* endpoints
```

## Production Deployment

### 1. Security Configuration
```bash
# Create production environment file
cp hfitd/.env.example .env.production

# Edit with secure values
nano .env.production
```

### 2. Use Environment File
```yaml
# In docker-compose.yml, replace environment section with:
env_file:
  - .env.production
```

### 3. External Networks (Optional)
```yaml
# Connect to existing database/redis
networks:
  default:
    external:
      name: your-existing-network
```

## Backup and Restore

### Redis Backup
```bash
# Backup Redis (user keys and JWT keys)
docker-compose exec redis redis-cli --rdb /data/backup.rdb

# Copy backup from container
docker cp $(docker-compose ps -q redis):/data/backup.rdb ./redis-backup.rdb
```

## Scaling and High Availability

### Load Balancer Configuration
```yaml
# Add to docker-compose.yml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - hfitd
```

### Multiple hfitd Instances
```yaml
# Scale the hfitd service
services:
  hfitd:
    # ... existing config
    deploy:
      replicas: 3
```

```bash
# Scale using docker-compose
docker-compose up -d --scale hfitd=3
```

## Troubleshooting

### Common Issues
```bash
# Check if services are healthy
docker-compose ps

# View service logs
docker-compose logs hfitd
docker-compose logs postgres
docker-compose logs redis

# Access container shell
docker-compose exec hfitd sh

# Test connectivity
docker-compose exec hfitd ping postgres
docker-compose exec hfitd ping redis
```

### Reset Everything
```bash
# Stop and remove everything
docker-compose down -v

# Remove images
docker-compose down --rmi all

# Start fresh
docker-compose up -d
```

## Environment Variables Reference

| Variable | Description | Example |
|----------|-------------|---------|
| `SERVER_ADDRESS` | HTTP server bind address | `:8080` |
| `DB_HOST` | **External** PostgreSQL hostname | `your-db-server.com` |
| `DB_PORT` | **External** PostgreSQL port | `5432` |
| `DB_USER` | **External** Database username | `your_user` |
| `DB_PASSWORD` | **External** Database password | `your_password` |
| `DB_NAME` | **External** Database name | `your_database` |
| `REDIS_URL` | Redis connection URL (provided by container) | `redis://:password@redis:6379` |
| `JWT_SECRET` | JWT signing secret | `very-secure-random-string` |
| `HFITD_SOCKET_PATH` | Admin socket path | `/tmp/hfitd/hfitd.sock` |

**Note:** The database configuration points to your existing database that contains the data you want to export. This is not provided by the Docker Compose stack.

## API Usage Examples

### Authentication Flow
```bash
# 1. Request challenge
CHALLENGE=$(curl -s -X POST http://localhost:8080/auth/challenge \
  -H "Content-Type: application/json" \
  -d '{"username":"admin@yourdomain.com"}' | jq -r '.challenge')

# 2. Sign challenge (client-side with private key)
# ... sign the challenge with your private key ...

# 3. Authenticate and get token
TOKEN=$(curl -s -X POST http://localhost:8080/auth/authenticate \
  -H "Content-Type: application/json" \
  -d "{
    \"username\":\"admin@yourdomain.com\",
    \"challenge\":\"$CHALLENGE\",
    \"signature\":\"$SIGNATURE\"
  }" | jq -r '.token')

# 4. Use token for API calls
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/db/dbs
```