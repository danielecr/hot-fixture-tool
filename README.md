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

- create a folder on your service repo: `mkdir hfit-data; cd hfit-data`
- keep your hot-export-package.yaml definition in files stored in the repo.
- run hfit to download data based on hot-export-package.yaml definition, into a target folder
- for each folder:
  - run import tool to read data from downloaded resource.
  - execute integration tests
  - cleanup (if needed)

A progressive workflow might fit better the integration test job. For this use this policy:

- define a base-export-package.yaml
- define as many hot-export-package.yaml as there are hot test cases.

Example of folder content:
~~~
└── hfit-data
    ├── .gitignore
    ├── base-export
    │   ├── db1.table1.create.sql
    │   ├── db1.table2.create.sql
    │   └── db1.table3.create.sql
    ├── base-export.yaml
    ├── use-case123
    │   ├── db1.table3.data.sql
    │   └── file-import-123.txt
    └── use-case123.yaml
~~~

**Importing data**

To use the `hfit` CLI to import data, use the `local-i` sub-command

`hfit local-i create my-docker.yaml extract-from base-export.yaml use-case123.yaml`

Open a text editor and fill the required data for (write) connection to your local dbms and filesystem provider.

Then run

`hfit local-i run-on my-docker.yaml base-export.yaml --no-download`

With `--no-download` option it imports data and just use the existing data.

Use a .gitignore in hfit/ folder like this:

~~~
*
!.gitignore
!*.yaml
~~~

to avoid the checkout of hot data.

As helper there is a convenient command:

`hfit prepare`

This creates `hfit-data/` folder and populates with .gitignore :

~~~
└── hfit-data
    └── .gitignore
~~~

## Definition

Service API:
- list DBS
- list tables
- export table definition
- export table data + where filter (with streaming NDJSON support)
- list accessible folder
- list files with filter (stream LDJSON)

### 🚀 Streaming Database Rows (New Feature)

Database table rows now support **streaming NDJSON format** for optimal performance with large datasets:

**Traditional JSON Response:**
```bash
curl -H 'Authorization: Bearer $JWT_TOKEN' \
     'http://localhost:8080/db/mysql/mydb/table/users/rows'
```

**Streaming NDJSON Response (Optimized):**
```bash
curl -H 'Authorization: Bearer $JWT_TOKEN' \
     -H 'Accept: application/x-json-stream' \
     'http://localhost:8080/db/mysql/mydb/table/users/rows'
```

**Advanced Filtering with SQL injection protection:**
```bash
# WHERE clause filtering
curl -H 'Accept: application/x-json-stream' \
     'http://localhost:8080/db/mysql/mydb/table/users/rows?filterpart=WHERE age > 25'

# Complex queries with ORDER BY and LIMIT
curl -H 'Accept: application/x-json-stream' \
     'http://localhost:8080/db/postgres/mydb/table/orders/rows?filterpart=WHERE status = "active" ORDER BY created_at DESC LIMIT 100'
```

**Key Benefits:**
- **O(1) Memory Usage**: Constant memory regardless of result size
- **Real-time Streaming**: First rows appear immediately  
- **SQL Injection Protection**: Safe filterpart parameter validation
- **Multi-DBMS Support**: Works with MySQL, PostgreSQL, and more
- **Enterprise Scalability**: Handles millions of rows efficiently

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
- ~~simple admin interface based on vanillaJs~~

## Export package format (WIP)

The command export-package accepts a .yaml file as a template definition for exporting hot data. The format is:

```yaml
hfitVersion: 1
templateName: usecase_data
ame: basedata_$1
prepare:
  - setVar: dataid
    from: input
    source: $1
  - setVar: usrId
    from: hot-data
    hdata:
        type: dbquery
        dbms: dbms_mysql1
        query: "SELECT usrId FROM dbname.datatable WHERE dataid=${dataid} ORDER BY utime LIMIT 1"
  - setVar: fBaseName
    from: hot-data
    hdata:
        type: volume
        volume: vol1
        glob: "*_{dataid}_{usrId}_*.txt"
        # take the first in mtime desc order:
        sort: "mtime|desc"
        # extract the first number of filename as fBaseName value
        regex_replace: "/([0-9]+).*/$1/"
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
- order of prepare is relevant

Once you defined this file, the common usage is to to store it in hfit-data/ folder of the repo. For this reason:

> hfit repo-prepare

Does create an hfit-data/ folder on the root of your git repo and write there some package template example.

hfit CLI should be used to create and populate this .yaml file.

Create the .yaml file:

> ./hfit pkg create base-data-package.yaml basedata

Execute package download and unpack into target folder (the target folder is named as the package name)

> ./hfit pkg downpack base-data-package.yaml

Manipulate by adding resource (this check if resource exists):

> ./hfit pkg add base-data-package.yaml <name> <type> <data>

Removing a resource:

> ./hfit pkg rm base-data-package.yaml <name> <data>

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

