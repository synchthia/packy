package storage

import (
	"encoding/json"
	"fmt"
	"os"
)

type Storage struct {
	Path string
}

type StorageAdapter interface {
	Load() error
	Save() error
}

func New(dirPath string) (*Storage, error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	fmt.Printf("[Storage] Using local storage: %s\n", dirPath)

	return &Storage{
		Path: dirPath,
	}, nil
}

func (s *Storage) Load(filePath string, entry interface{}) (bool, error) {
	exists := false
	if _, err := os.Stat(s.Path + "/" + filePath); os.IsNotExist(err) {
		s.Save(filePath, entry)
	} else if err == nil {
		exists = true
	}

	b, err := os.ReadFile(s.Path + "/" + filePath)
	if err != nil {
		return exists, err
	}
	if err := json.Unmarshal(b, entry); err != nil {
		return exists, err
	}
	return exists, nil
}

func (s *Storage) Save(filePath string, entry interface{}) error {
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.Path+"/"+filePath, b, os.ModePerm); err != nil {
		return err
	}
	return nil
}
