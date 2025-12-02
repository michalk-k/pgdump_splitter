# CHANGELOG
# 1.2.1
* Make possible to pass hash for restrict/unrestrict

# 1.2.0
* skip \restrict and \unrestrict lines introduced by postgresql v17.6

# 1.1.0
* removed extracting documentation from function source code (`-doc` parameter)

## 1.0.6
* fix: no roles copied for the last database in the cluster if -mc switch is enabled

## 1.0.5
* fix: extract DOCU from procedures

## 1.0.4
* fix: NOT VALID check constraints are not included to table DDL. Appended now with mode=custom

## 1.0.3
* fix: improperly parsed meta for procedure's ACLs

## 1.0.2
* fix: empty roles when using -mc switch

## 1.0.1
* fix: missing hashed arguments for stored procedures

## 1.0.0
* initial release
