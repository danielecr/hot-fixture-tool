#!/bin/bash

# Generate RSA key pair for Hot Fixture Tool authentication

echo "Generating RSA key pair for Hot Fixture Tool..."

# Generate private key
openssl genrsa -out private_key.pem 2048
echo "✓ Generated private_key.pem"

# Generate public key
openssl rsa -in private_key.pem -pubout -out public_key.pem
echo "✓ Generated public_key.pem"

echo ""
echo "Keys generated successfully!"
echo ""
echo "To use with your .env file:"
echo "1. Copy the content of public_key.pem"
echo "2. Set PUBLIC_KEY_PEM in your .env file"
echo ""
echo "Example:"
echo 'PUBLIC_KEY_PEM="$(cat public_key.pem)"'
echo ""
echo "⚠️  Keep private_key.pem secure and never commit it to version control!"
echo "⚠️  The client application will need private_key.pem to authenticate"