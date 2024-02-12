# Motivation
The main reason why this program has been created is to quickly dump databases into files, representing different database objects.

Why not a single file? \
The result has to be versioned (ie in GIT) and then worked on by teams. A single file means merging conflicts. The more files the structure is split into, the fewer merging conflicts. \
Also, working with small files, and comparing their content (ie in GIT) is more comfortable than with a single big one.

# Features
1. May use `pg_dump` directly
2. Alternatively may parse a dump file, generated earlier by pg_dump
3. Dumps each db object to separate file
4. Allows grouping some objects into a single file (ie table, its acls, comments etc)
5. Filenames of files containing functions are shortened to not exceed the maximum file length allowed by the filesystem/os
   
