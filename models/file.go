package models

type Content struct {
	// Name - File name
	Name string `json:"name"`

	// Local Path - local path without namespace
	LocalPath string `json:"local_path"`

	// Path - File path
	Path string `json:"path"`

	// Hash - file hash (ex. etag)
	Hash string `json:"hash"`
}
