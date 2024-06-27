package dbobject

import (
	"reflect"
	"strings"
	"testing"
)

//

func TestArgumentsEncoding(t *testing.T) {

	want := ""
	got := ""

	want = "quote_empty-02103c"
	got = generateFuncFilename("quote_empty", "character varying, integer[]")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "quote_empty"
	got = generateFuncFilename("quote_empty", "")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

}

func TestGetFuncIdentParts(t *testing.T) {

	want_fc, want_args := "send_email", "text, public.hstore, text, text, text"
	fc, args := getFuncIdentParts("send_email(text, public.hstore, text, text, text)")

	if want_fc != fc {
		t.Errorf("got %s, wants %s", fc, want_fc)
	}

	if want_args != args {
		t.Errorf("got %s, wants %s", args, want_args)
	}

}

func TestRemoveArgumentsFromFunction(t *testing.T) {

	want := ""
	got := ""

	want = "public.hstore"
	got = NormalizeFunctionIdentArgs("public.hstore")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "text, text, text[], text[]"
	got = NormalizeFunctionIdentArgs("_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[]")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "text, text, text, text, integer"
	got = NormalizeFunctionIdentArgs("text, text, text, text, integer")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "boolean, oid, oid, boolean, bigint"
	got = NormalizeFunctionIdentArgs("showtext boolean, OUT userid oid, OUT dbid oid, OUT toplevel boolean, OUT queryid bigint")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "text, public.hstore, text, text, text"
	got = NormalizeFunctionIdentArgs("text, public.hstore, text, text, text")

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	//	"Name: send_email(text, public.hstore, text, text, text); Type: FUNCTION; Schema: communication_api; Owner: sazky"

}

func TestFuncionPath1_custom(t *testing.T) {

	dbo := DbObject{
		Name:    "FUNCTION avals(public.hstore)",
		ObjType: "ACL",
		Schema:  "public",
	}

	dbo.Paths = DbObjPath{
		Rootpath: "/root/",
		IsCustom: true}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{Schema: "public", Name: "avals(public.hstore)", ObjType: "ACL", ObjSubtype: "FUNCTION", ObjSubName: "avals(public.hstore)",
		Paths: DbObjPath{Rootpath: "/root/", NameForFile: "avals-c66339", FullPath: "/root/public/function/avals-c66339.sql", IsCustom: true},
	}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 1 failed")
	}

}

func TestFuncionPath1_orig(t *testing.T) {

	dbo := DbObject{
		Name:    "FUNCTION avals(public.hstore)",
		ObjType: "ACL",
		Schema:  "public",
		Paths: DbObjPath{
			Rootpath: "/root/",
			IsCustom: false},
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{Schema: "public", Name: "avals(public.hstore)", ObjType: "ACL", ObjSubtype: "FUNCTION", ObjSubName: "avals(public.hstore)",
		Paths: DbObjPath{Rootpath: "/root/", NameForFile: "avals-c66339", FullPath: "/root/public/ACL/avals-c66339.sql", IsCustom: false},
	}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 1 failed")
	}

}

func TestFuncionPath2(t *testing.T) {

	dbo := DbObject{
		Name:    "FUNCTION column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])",
		ObjType: "ACL",
		Schema:  "public",
		Paths:   DbObjPath{Rootpath: "/root/", IsCustom: true},
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{
		Schema:     "public",
		Name:       "column_names(text, text, text[], text[])",
		ObjType:    "ACL",
		ObjSubtype: "FUNCTION",
		ObjSubName: "column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])",
		Content:    strings.Builder{},
		Paths: DbObjPath{
			Rootpath:    "/root/",
			NameForFile: "column_names-a09c34",
			FullPath:    "/root/public/function/column_names-a09c34.sql",
			IsCustom:    true,
		},
	}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 2 failed")
	}

}

func TestFuncionPath2_orig(t *testing.T) {

	dbo := DbObject{
		Name:    "FUNCTION column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])",
		ObjType: "ACL",
		Schema:  "public",
		Paths:   DbObjPath{Rootpath: "/root/", IsCustom: false},
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{
		Schema:     "public",
		Name:       "column_names(text, text, text[], text[])",
		ObjType:    "ACL",
		ObjSubtype: "FUNCTION",
		ObjSubName: "column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])",
		Paths: DbObjPath{
			Rootpath:    "/root/",
			NameForFile: "column_names-a09c34",
			FullPath:    "/root/public/ACL/column_names-a09c34.sql",
			IsCustom:    false,
		},
	}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 2 failed")
	}

}

func TestForeignDataWrapperAcl(t *testing.T) {

	dbo := DbObject{

		Name:    "FOREIGN DATA WRAPPER dblink_fdw",
		ObjType: "ACL",
		Schema:  "-",
		Paths: DbObjPath{
			Rootpath: "/root/",
			IsCustom: true,
		},
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{

		Schema:     "-",
		Name:       "FOREIGN DATA WRAPPER dblink_fdw",
		ObjType:    "ACL",
		ObjSubtype: "FOREIGN DATA WRAPPER",
		ObjSubName: "dblink_fdw",
		Paths: DbObjPath{
			Rootpath:    "/root/",
			NameForFile: "dblink_fdw",
			FullPath:    "/root/-/foreign data wrapper/dblink_fdw.sql",
			IsCustom:    true,
		},
	}
	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test TestRootObjects() failed")
	}
}

func TestDatabaseAclPath(t *testing.T) {

	dbo := DbObject{

		Schema:     "-",
		Name:       "DATABASE betsys",
		ObjType:    "ACL",
		ObjSubtype: "DATABASE",
		ObjSubName: "betsys",
		Database:   "betsys",
		AclFiles:   true,
		Paths: DbObjPath{
			Rootpath: "/root/",
			IsCustom: true,
		},
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := "/root/betsys/-/database/betsys.acl.sql"

	if want != dbo.Paths.FullPath {
		t.Errorf("test TestDatabaseAclPath() failed")
	}
}
