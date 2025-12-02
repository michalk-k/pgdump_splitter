package dbobject

// Structure handling program runtime configuration.
// Values are comming from command line arguments.
type Config struct {
	Mode     string
	Dest     string
	NoDb     bool
	ExDb     string
	ExOT     string
	WlDb     string
	MvRl     bool
	File     string
	Docu     string
	BufS     int
	Cln      bool
	Quiet    bool
	AclFiles bool
	Restrict string
}
