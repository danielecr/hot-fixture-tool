# Hot Fixture Tool (hfit)

[![Sponsor](https://img.shields.io/badge/sponsor-danielecr-brightgreen?logo=github&style=for-the-badge)](https://github.com/sponsors/danielecr) | [Donate](https://paypal.me/danielecru) | [ko-fi](https://ko-fi.com/danielecruciani)

Hot Fixture Tool is compound of two pieces:

- server (hfitd) installed on server with access hot db and fs data
- client (hfit) is a CLI interface to the server

## Ratio and use cases

Development of DataBase centric service involve unit tests and integration tests definition. Unit tests are used to keep the code clean and to lower cyclomatic and other code metrics. Integration tests are used to check the effective correctnes of the service.

This service and tool targets the integration test definition.

By using hfit, a developer can:

- define resources as DDL, db rows, and file to be ready to execute integration test locally
- run the integration tests against a real set of data
- **retrieve hot data for hot fixing of bug a bug**
- automate the retrievement of hot data to execute all integration tests and avoid regressions

Most of the time, running a dbms locally is not resource consuming as it is the repeated cycle of "blinded development".

The most common case is the deal with legacy application that need to be ported, refactored, adapted.

An application become "legacy" after 5 or 6 year of development, so this tool is mostly targeted at successful projects with business logic, that one want to keep alive and current, want to update to newer execution engine, compiler or interpreter, want to change the protocol exposed, etc.

## Usage

- keep your hot-export-package.yaml definition in files stored in the repo.
- run hfit to download data based on hot-export-package.yaml definition, into a target folder
- for each folder:
  - run import tool to read data from downloaded resource.
  - execute integration tests
  - cleanup (if needed)

A progressive workflow might fit better the integration test job. For this use this policy:

- define a base-export-package.yaml
- define as many hot-export-package.yaml as there are hot test cases.

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

## Export package format (WIP)

The command export-package accepts a .yaml file with definition for hot data exports. The format is:

```yaml
name: basedata
exports:
  dbcreate.sql:
    type: dbcreate
    data:
      dbms: dbms_mysql1
      tablelist:
        - dbname1
        - dbname2
  tablegroup1.create.sql:
    type: table-create
    data:
      dbms: dbms_mysql1
      tablelist:
        - dbname1.table1
        - dbname2.table2
        - dbname1.tablex
      option: <dropcreate|ifnotexists>
  tabledata1.data.sql:
    type: table-data
    data:
      dbms: dbms_mysql1
      table: dbname1.table1
      filter: WHERE key<34 AND key>12
  target-filename.txt:
    type: file
    data:
      volume: datavol1
      path: relative/path/to/file
```

NOTES:
- order of retrievement is not important

hfit CLI should be used to create and populate this .yaml file.

Create the .yaml file:

> ./hfit pkg create base-data-package.yaml basedata

Execute package download and unpack into target folder (the target folder is named as the package name)

> ./hfit pkg downpack base-data-package.yaml

Manipulate by adding resource (this check if resource exists):

> ./hfit pkg add basedata <name> <type> <data>

Removing a resource:

> ./hfit pkg rm basedata <name> <data>

Edit (option/filter/tablename)

## Plan

First release has no ACL control, all users read access to all resources

## Quick Start with Docker

The fastest way to get started is using Docker:

```bash
# Start the complete stack (PostgreSQL + Redis + hfitd)
make demo

# Or manually:
docker-compose up -d
make adduser EMAIL=admin@yourdomain.com KEY=your_public_key.pem
```

Visit `http://localhost:8080` and see [DOCKER.md](DOCKER.md) for complete deployment guide.

## Looking for Contributors & Feedback

This project is **source-available** - you can inspect the code, contribute improvements, and use it for personal/educational purposes. I'm actively seeking:

- **Contributors** to help improve the codebase
- **Feedback** from users who find this tool useful
- **Sponsorship** or commercial partnerships
- **Real-world use cases** and feature requests

If you find this project valuable or want to use it commercially, please reach out! I'm open to discussing licensing, sponsorship, or collaboration opportunities.

## License

This project is licensed under a custom source-available license. You may:
- Inspect and study the source code
- Contribute improvements and bug fixes
- Use for personal, non-commercial purposes

You may NOT:
- Use commercially without permission
- Distribute without permission

See [LICENSE](LICENSE) for full details. For commercial use, please contact me to discuss licensing options.

## Contributing

Contributions are welcome! Please feel free to:
- Report bugs and issues
- Suggest new features
- Submit pull requests
- Provide feedback and suggestions

By contributing, you help make this tool better while allowing me to explore sustainable development models.

