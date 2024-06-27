# Motivation
This utility is designed with a specific purpose in mind: to facilitate the creation of a file structure that mirrors a database dump, enabling efficient versioning within a GIT repository.

Why not simply one large file?\
The decision to structure the data across multiple files stems from the necessity to minimize merging conflicts when collaborating across teams. By breaking down the database dump into smaller, more manageable files, the likelihood of encountering excessive merging conflicts is significantly reduced. Additionally, the granularity provided by smaller files enhances the ease of comparing and managing content within GIT, making the collaborative process smoother and more streamlined.

# Features
1. Supports SQL dumps created by `pg_dump` and `pg_dumpall`
2. Can use the dumped file or direct stream through a system pipe
3. Dumps each db object to separate file
5. Allows grouping of related objects into a single file (ie table together with its acls, comments, column comments, defaults etc)
6. Allows to move role definitions, privileges and config to the substructure of each database
7. Files containing functions have filenames shortened to avoid exceeding the maximum file length allowed by the filesystem/os

## Modes
The utility provides two modes of reflecting dump stems on filesystem objects (files).
### origin
In this mode, the resulting structure of files and their names exactly reflects what is found in dump files created by `pg_dump` or `pg_dumpall`. For instance:
* every index, constraint, trigger, and acls are stored in separate files
* comment and acl file names are prefixed by the object they belong to. For example `...schema_name/ACL/TABLE tablename.sql` or `...schema_name/COMMENT/COUMN tablename.columnname.sql`
### custom
The custom mode is an attempt to aggregate related objects in single files.
* indexes, constraints, triggers, as well as comments of the table and their columns are appended to the table sql
* ACLs of all objects are appended to their respective object files. Optionally might be stored to separate files named after the original object: `original_object.acl.sql`
* settings of databases are appended to their respective database ddl files
* tables being published are appended to respective publication ddl files
* inheritance of roles, as well as their settings, are appended to roles ddl

  On top of that subdirectories organizing object types are converted to lowercase.

# Limitations
*1.*
The program scans dump files line by line executing regular expression matching against them to extract blocks of code. For this reason, using text patterns listed below in a source code of any function may confuse the utility.

`\connect some_string`\
`-- PostgreSQL database dump`\
`-- PostgreSQL database dump complete`\
`-- Name: some_string; Type: some_string; Schema: some_string;`\
`-- Data for Name: some_string; Type: some_string; Schema: some_string;`

*2.*
The utility is not designed to accommodate databases with object names containing space characters.
The utility relies on metadata extracted from comments in SQL dumps to identify database objects. Regrettably, the accuracy of these metadata often suffers when object names include spaces, thereby rendering proper data segmentation impossible.

*3.*
The utility is not tested for object names requiring double-quoting. Double quoting is required if the object name consists of upper case characters, national and special characters  characters, national characters, upper case 
   
# Usage
`pgdump_splitter {options} -f {dump_file}`\
or\
`{pg_dump|pg_dumpall} --schema-only ... | pgdump_splitter {options}`

Expected data has to be compliant with the `plain` format of an output generated by pg_drump or pg_dumpall. See the respective tools documentation for details.

Mentioned --schema-only is suggested since `pgdump_splitter` skips dumped data anyway.


Command-line options listed below, control the `pgdump_splitter` utility. Because of using Golang built-in command line parser, single and double hyphens are accepted for every option. Option values might be passed with the use of an `equal` or `space` character.

`-mode=modename`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; The mode of dumping db objects. origin - files are organized as present in the database dump. custom - reorganizes db objects joining related ones into a single file. The default is `custom`
     
`-f=path/to/source/file`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; Path to dump generated by `pg_dump` or `pg_dumpall`. If omited the program expects data on stdin via system pipe.

`-dst=path/to/destination/directory`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; Location where structures will be dumped to.

`-clean`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;R emove any content from destination directory.

`-ndb`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; No db name in destination path. Setting it to true for a dump containing multiple databases is meaningless.

`-blacklist-db=regular.expression`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; Regular expression pattern allowing to skip extraction of matching databases. Useful in case of processing dump files. Ignored if `-whitelist-db` is used. In case of using a pipe from `pg_dumpall`, it's better to exclude databases using `pd_dumpall` option. Default is `^(template|postgres)`

`-whitelist-db=regular.expression`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Regular expression pattern allowing to whitelist databases. If set, only databases matching this expression are processed. Thus blacklist is not applicable.

`-mc`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Copy files containing role-related definitions into each database subdirectory. Otherwise they will be found in '{dst}/-/' subdirectory 

`-buffer=number`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Set up maximum buffer size if your dump contains data not fitting the scanner. The default is `1048576`

`-quiet`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Suppress all messages printed to standard output. Errors are still printed to err output.

`-aclfiles`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Applicable or mode=custom only. Makes GRANTs be output to separate files suffixed with `.acl.sql`, ie `table_name.acl.sql`. Otherwise, acls are appended to related object files.

`-version`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Print the pgdump_spritter version and exit.


**Examples**

`pgdump_splitter -mode origin -f dbdump.sql -dst /path/to/resulting/structure/`\
or\
`cat dbdump.sql | pgdump_splitter -mode origin -s /path/to/resulting/structure/`

Creates the result from `dbdump.sql` file generated by `pg_dump` or `pg_dumpall` earlier. Result files are organized into file structures proposed by the dump commands.

`pg_dump --schema-only ... | pgdump_splitter -mode custom -mc -dst /path/to/resulting/structure/`\
or\
`pg_dumpall --schema-only ... | pgdump_splitter -mode custom -mc -dst /path/to/resulting/structure/`

Creates the result from data streamed directly from `pg_dump` or `pg_dumpall` connected to a given database. Result files are organized in a way, aggregating related objects into single files (ie objects together with their ACLs). Roles definitions, their inheritance and configuration are moved into `{database_name}/-/` subdirectory

