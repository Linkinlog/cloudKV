package store

import (
	"errors"
	"sync"
)

var ErrNoSuchKey = errors.New("no such key")

type KeyValueStore struct {
	lock *sync.Mutex
	m    map[string]string
}

func New() *KeyValueStore {
	m := make(map[string]string)
	lock := &sync.Mutex{}

	return &KeyValueStore{lock: lock, m: m}
}

func (k *KeyValueStore) Put(key, value string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	k.m[key] = value
	return nil
}

func (k *KeyValueStore) Delete(key string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	delete(k.m, key)

	return nil
}

func (k *KeyValueStore) Get(key string) (string, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	value, ok := k.m[key]
	if !ok {
		return "", ErrNoSuchKey
	}

	return value, nil
}
