package dbobject

// Structure handling program runtime configuration.
// Values are comming from command line arguments.
type Config struct {
	Mode string
	Dest string
	NoDb bool
	ExDb string
	MvRl bool
	File string
	Docu string
	BufS int
}
