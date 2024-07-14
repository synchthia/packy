package models

type Content struct {
	// Name - File name
	Name string `json:"name"`

	// Path - File path
	Path string `json:"path"`

	// Hash - file hash (ex. etag)
	Hash string `json:"hash"`
}
