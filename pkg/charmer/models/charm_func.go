package models

type CharmFunc struct {
	Name        string
	Doc         string
	Execute     interface{}
	Path        string
	Title       string
	Description string
}
