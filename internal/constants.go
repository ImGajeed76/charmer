package internal

type uiTheme struct {
	PrimaryColor   string
	SecondaryColor string
	ErrorColor     string
	TertiaryColor  string
}

var Theme = uiTheme{
	PrimaryColor:   "75",      // Brighter blue
	SecondaryColor: "#ccc",    // Lighter gray for better readability
	ErrorColor:     "#FF5F5F", // Red for errors
	TertiaryColor:  "#666666", // Orange for warnings
}
