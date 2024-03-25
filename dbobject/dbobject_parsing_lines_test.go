package dbobject

import (
	"testing"
)

func TestDatabaseLine(t *testing.T) {

	want := ""
	got := ""

	want = "database_name"
	src := `\connect database_name`
	got = InitDatabaseFromLine(&src)

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}

	want = "db with space"
	src = `\connect -reuse-previous=on "dbname='db with space'"`
	got = InitDatabaseFromLine(&src)

	if want != got {
		t.Errorf("got %s, wants %s", got, want)
	}
}

func TestEndOfCluster(t *testing.T) {

	var want int
	var got int

	want = 0
	src := "some text"
	got = MatchUsersAndDatabasesStart(&src)

	if want == got {
		t.Errorf("got %d, wants %d", got, want)
	}

	want = 1
	src = "-- User Configurations"
	got = MatchUsersAndDatabasesStart(&src)

	if want != got {
		t.Errorf("got %d, wants %d", got, want)
	}

	want = 2
	src = "-- Databases"
	got = MatchUsersAndDatabasesStart(&src)

	if want != got {
		t.Errorf("got %d, wants %d", got, want)
	}

}

func TestMatchDbStartEnd(t *testing.T) {

	var want int
	var got int

	want = -1
	src := "some text"
	got = MatchDbStartEnd(&src)

	if want != got {
		t.Errorf("got %d, wants %d", got, want)
	}

	want = 0
	src = "-- PostgreSQL database dump complete"
	got = MatchDbStartEnd(&src)

	if want != got {
		t.Errorf("got %d, wants %d", got, want)
	}

	want = 2
	src = "-- PostgreSQL database dump"
	got = MatchDbStartEnd(&src)

	if want != got {
		t.Errorf("got %d, wants %d", got, want)
	}

}

func TestOfInitRoleObjFromLine(t *testing.T) {

	var cfg Config
	var dbo DbObject

	src := `-- User Config "app_olddataremoval"`
	dbo = *InitRoleObjFromLine(&src, &cfg, "db_name")

	if dbo.ObjType != "ROLE" {
		t.Errorf("test of `User Config` failed")
	}

	src = `-- Role memberships`
	dbo = *InitRoleObjFromLine(&src, &cfg, "db_name")

	if dbo.ObjType != "ROLE" {
		t.Errorf("test of `Role memberships` failed")
	}

	src = `-- Roles`
	dbo = *InitRoleObjFromLine(&src, &cfg, "db_name")

	if dbo.ObjType != "ROLE" {
		t.Errorf("test of `Roles` failed")
	}

}

func TestOfInitCommonObjFromLine(t *testing.T) {

	var cfg Config
	var dbname = "db_name"
	var dbo DbObject

	src := "-- Name: TABLE q_tickets_client_notifications_betsys; Type: ACL; Schema: sql_queues; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	if !(dbo.ObjType == "ACL" && dbo.Name == "TABLE q_tickets_client_notifications_betsys" && dbo.Schema == "sql_queues" && dbo.Database == dbname) {
		t.Errorf("test 1 failed")
	}

	src = "-- Name: DEFAULT PRIVILEGES FOR SEQUENCES; Type: DEFAULT ACL; Schema: sql_queues; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	if !(dbo.ObjType == "DEFAULT ACL" && dbo.Name == "DEFAULT PRIVILEGES FOR SEQUENCES" && dbo.Schema == "sql_queues" && dbo.Database == dbname) {
		t.Errorf("test 2 failed")
	}

	src = "-- Name: lst_permissions; Type: TABLE; Schema: app_permissions; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	if !(dbo.ObjType == "TABLE" && dbo.Name == "lst_permissions" && dbo.Schema == "app_permissions" && dbo.Database == dbname) {
		t.Errorf("test 2 failed")
	}

	src = "-- Name: quote_empty(character varying, integer[]); Type: FUNCTION; Schema: utl; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	if !(dbo.ObjType == "FUNCTION" && dbo.Name == "quote_empty(character varying, integer[])" && dbo.Schema == "utl" && dbo.Database == dbname) {
		t.Errorf("test 2 failed")
	}

}

func TestOfCompleteObjProcessing_Function(t *testing.T) {

	var cfg Config
	var dbname = "db_name"
	var dbo DbObject

	src := "-- Name: quote_empty(character varying, integer[]); Type: FUNCTION; Schema: utl; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	dbo.generateDestinationPath()

	if !(dbo.ObjType == "FUNCTION" && dbo.Name == "quote_empty(character varying, integer[])" && dbo.Schema == "utl" && dbo.Database == dbname) {
		t.Errorf("test 2 failed")
	}

	src = "-- Name: quote_empty(character varying, integer[]); Type: FUNCTION; Schema: utl; Owner: sazky"
	dbo = *InitCommonObjFromLine(&src, &cfg, dbname)

	dbo.generateDestinationPath()

	if !(dbo.ObjType == "FUNCTION" && dbo.Name == "quote_empty(character varying, integer[])" && dbo.Schema == "utl" && dbo.Database == dbname) {
		t.Errorf("test 2 failed")
	}

}
