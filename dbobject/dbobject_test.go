package dbobject

import (
	"reflect"
	"testing"
)

func TestRemoveArgumentsFromFunction(t *testing.T) {

	want := ""
	got := ""

	want = "avals(public.hstore)"
	got = removeArgNamesFromFunctionIdent("avals(public.hstore)")
	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "column_names(text, text, text[], text[])"
	got = removeArgNamesFromFunctionIdent("column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])")
	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "connectby(text, text, text, text, integer)"
	got = removeArgNamesFromFunctionIdent("connectby(text, text, text, text, integer)")
	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "pg_stat_statements(boolean, oid, oid, boolean, bigint))"
	got = removeArgNamesFromFunctionIdent("pg_stat_statements(showtext boolean, OUT userid oid, OUT dbid oid, OUT toplevel boolean, OUT queryid bigint))")
	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}
}

func TestFuncionPath1(t *testing.T) {

	dbo := DbObject{
		Rootpath: "/root/",
		Name:     "FUNCTION avals(public.hstore)",
		ObjType:  "ACL",
		Schema:   "public",
		IsCustom: true,
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{Rootpath: "/root/", Schema: "public", Name: "FUNCTION avals(public.hstore)", ObjType: "ACL", ObjSubtype: "FUNCTION", ObjSubName: "avals-c66339", FullPath: "/root/public/functions/avals-c66339.sql", Content: "", IsCustom: true}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 1 failed")
	}

}

func TestFuncionPath2(t *testing.T) {

	dbo := DbObject{
		Rootpath: "/root/",
		Name:     "FUNCTION column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])",
		ObjType:  "ACL",
		Schema:   "public",
		IsCustom: true,
	}

	dbo.normalizeDbObject()
	dbo.generateDestinationPath()

	want := DbObject{Rootpath: "/root/", Schema: "public", Name: "FUNCTION column_names(_schema_name text, _table_name text, _not_in_column_names text[], _not_in_data_types text[])", ObjType: "ACL", ObjSubtype: "FUNCTION", ObjSubName: "column_names-a09c34", FullPath: "/root/public/functions/column_names-a09c34.sql", Content: "", IsCustom: true}

	if !reflect.DeepEqual(dbo, want) {
		t.Errorf("test 2 failed")
	}

}
