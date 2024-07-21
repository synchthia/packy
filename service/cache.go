package service

import "github.com/synchthia/packy/storage"

type CacheService struct {
	store *storage.Storage
	Cache *Cache
}

type Cache struct {
	Files map[string]*CachedFile `json:"files,omitempty"`
}

type CachedFile struct {
	Hash string `json:"hash"`
}

const fileName = "packy-cache.json"

func InitCache(filePath string) (*CacheService, error) {
	cache := &Cache{}
	if cache.Files == nil {
		cache.Files = make(map[string]*CachedFile)
	}

	store, err := storage.New(filePath)
	if err != nil {
		return nil, err
	}

	if _, err := store.Load(fileName, cache); err != nil {
		return nil, err
	}

	return &CacheService{
		store: store,
		Cache: cache,
	}, nil
}

func (c *CacheService) Save() error {
	return c.store.Save(fileName, c.Cache)
}
