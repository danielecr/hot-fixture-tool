# Adopt package template

The yaml format for hfit package does not fit some requirements. For example:
- user want to query for a single table to extract an id and then reuse this for others table-data rows extraction
- user want to reuse variable(s) for file download
- user want to use external variable to create a case-specific package, like usecase-123-$uid-$datais.tar.gz
- user want that some step just calculate variables.
So the yaml format must support variables of 2 types:
- external variable
- internal (calculated) variable
To calculate a variable, it is important that the order of execution is respected, but object has just a property:value map that generally is ordered alphanumerically. On other hand, object warrant a key is not duplicated.
For this reason I think a better format one with a "prepare" key, that is used for defining all variables, and "prepare" is an array of object.
I changed the yaml example in README.md adding a prepare property.
But also I want that yaml that accept external variable are called package-template (pkg-tmpl).
Now, I want to remove pkg sub command from client, the developer is expected to edit the yaml to define the data export, including prepare.
For downloading a package the command should be

hfit pkg-tmpl exec export-tmpl.yaml <var1> <var2>

I will go on with specification later.
As first step I want you to check the yaml format, and confirm me that this is a good and correct way to use array of object in prepare
Please, do not produce code if there are no synthax improvement to be suggested on yaml format (i.e. do not produce code if it is not yaml)

The format:

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
        glob: *_{dataid}_{usrId}_*.txt
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
