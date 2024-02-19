# Motivation
This utility is designed with a specific purpose in mind: to facilitate the creation of a file structure that mirrors a database dump, enabling efficient versioning within a GIT repository.

Why not simply one large file?\
The decision to structure the data across multiple files stems from the necessity to minimize merging conflicts when collaborating across teams. By breaking down the database dump into smaller, more manageable files, the likelihood of encountering excessive merging conflicts is significantly reduced. Additionally, the granularity provided by smaller files enhances the ease of comparing and managing content within GIT, making the collaborative process smoother and more streamlined.

# Features
1. Supports SQL dumps created by `pg_dump` and `pg_dumpall`
2. Can use dumped file or direct stream through system pipe
3. Dumps each db object to separate file
4. Provides 2 modes of splitting the dumps
5. Allows grouping of related objects into a single file (ie table together with its acls, comments, column comments, defaults etc)
6. Allows to move roles definitions to substructure of each database
7. Files containing functions have filenames shortened to avoid exceeding the maximum file length allowed by the filesystem/os
8. Extracts documentation found in functions code
   
# Usage
`pgdump_splitter {options} -f {dump_file}`\
or\
`{pg_dump|pg_dumpall} ... | pgdump_splitter {options}`

Expected data has to be compliant with the `plain` format of an output generated by pg_drump or pg_dumpall. See respective tools documentation for details.\

The following command-line options control the `pgdump_splitter` utility:

`-mode`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;The mode of dumping db objects. origin - files are organized as present in the database dump. custom - reorganizes db objects storing related ones into single file
     
`-f`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;path to dump generated by `pg_dump` or `pg_dumpall`. If omited the program will expect data on stdin via system pipe.

`-dst`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Location where structures will be dumped to.

`-ndb`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;No db name in destination path. Setting it to true if multiple databases are dumped at once is meaningless.

`-exdb`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Regular expression pattern allowing to skip extraction of matching databases. Usefull in case of processing dump files. In case of using a pipe from `pg_dumpall`, exclude them using `pd_dumpall` switch

`-mc`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Move dump of roles into each database subdirectory. Otherwise they will be found in '{dst}/-/' subdirectory

`-doc`

&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Regular expression used to extract docuementation out of function source code into separate .md files. Default is `DOCU(.*)DOCU`


**Examples**

`pgdump_splitter -mode origin -f dbdump.sql -dst /path/to/resulting/structure/`\
or\
`cat dbdump.sql | pgdump_splitter -mode origin -s /path/to/resulting/structure/`

Creates the result from `dbdump.sql` file generated by `pg_dump` or `pg_dumpall` earlier. Result files are organized into file structures proposed by the dump commands.

`pg_dump ... | pgdump_splitter -mode custom -mc -dst /path/to/resulting/structure/`\
or\
`pg_dumpall ... | pgdump_splitter -mode custom -mc -dst /path/to/resulting/structure/`

Creates the result from data streamed directly from `pg_dump` or `pg_dumpall` connected to a given database. Result files are organized in a way, aggregating related objects into single files (ie objects together with their ACLs). Roles definitions, their inheritance and configuration are moved into `{database_name}/-/` subdirectory

