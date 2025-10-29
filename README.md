# Hot Fixture Tool (hfit)

Hot Fixture Tool is built as a client-server architecture:

- server (hfitd) can be deployed as docker image and can access hot db and fs data
- client (hfit) is a CLI interface to the server

## Definition

Service API:
- list DBS
- list tables
- export table definition
- export table data + where filter
- list accessible folder
- list files with filter (stream LDJSON)

Client:
- interactively run all API provided by service 
- define JSON for export package with parameters
- modify export JSON 
- execute export package.

Features:
- service ACL per user
- user auth list
- user authentication by key-pair, challenge, and jwt
- storage by redis
- admin user- public key
- simple admin interface based on vanillaJs

## Plan

First release has no ACL control, all users has access to all resources

## � Quick Start with Docker

The fastest way to get started is using Docker:

```bash
# Start the complete stack (PostgreSQL + Redis + hfitd)
make demo

# Or manually:
docker-compose up -d
make adduser EMAIL=admin@yourdomain.com KEY=your_public_key.pem
```

Visit `http://localhost:8080` and see [DOCKER.md](DOCKER.md) for complete deployment guide.

## �🔍 Looking for Contributors & Feedback

This project is **source-available** - you can inspect the code, contribute improvements, and use it for personal/educational purposes. I'm actively seeking:

- **Contributors** to help improve the codebase
- **Feedback** from users who find this tool useful
- **Sponsorship** or commercial partnerships
- **Real-world use cases** and feature requests

If you find this project valuable or want to use it commercially, please reach out! I'm open to discussing licensing, sponsorship, or collaboration opportunities.

## License

This project is licensed under a custom source-available license. You may:
- ✅ Inspect and study the source code
- ✅ Contribute improvements and bug fixes
- ✅ Use for personal, non-commercial purposes
- ❌ Use commercially without permission
- ❌ Distribute without permission

See [LICENSE](LICENSE) for full details. For commercial use, please contact me to discuss licensing options.

## Contributing

Contributions are welcome! Please feel free to:
- Report bugs and issues
- Suggest new features
- Submit pull requests
- Provide feedback and suggestions

By contributing, you help make this tool better while allowing me to explore sustainable development models.

