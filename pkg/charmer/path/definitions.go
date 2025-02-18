package path

type Path struct {
	path     string
	isSftp   bool
	host     string
	port     string
	username string
	password string
}

type SFTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}
