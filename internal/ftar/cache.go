package ftar

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
)

const (
	MaxFtars = 18
	IconPath = "/tmp/icons"
	FixedOut = "/tmp/ftars.fixed"

	defaultGroup = "human"
)

var FtarsIn = "/ftars"

func NewCache() *Cache {
	c := &Cache{
		ftars: make(map[string]*Ftar),
	}
	os.MkdirAll(IconPath, 0755)
	os.MkdirAll(FixedOut, 0755)

	de, err := os.ReadDir(FtarsIn)
	if err != nil {
		log.Println("Error reading fixed ftars", err)
		return c
	}

	log.Println("fixed qty", len(de))
	for _, e := range de {
		log.Println("reading subdir", e.Name())
		c.readSubdir(FtarsIn, e.Name())
	}

	return c
}
func (c *Cache) readSubdir(fullPath string, group string) {
	de, err := os.ReadDir(path.Join(fullPath, group))
	if err != nil {
		return
	}

	log.Println("fixed qty", len(de))
	for _, e := range de {
		log.Println(e.Name())
		f := &Ftar{
			File:  path.Join(fullPath, group, e.Name()),
			Group: group,
		}
		f.Key()
		if err := f.CopyAndFetchPng(FixedOut); err != nil {
			log.Println("fixed loading", err)
		}
		c.ftars[f.Id] = f
	}
}

type Cache struct {
	mu    sync.RWMutex
	ftars map[string]*Ftar
}

func (c *Cache) AddFtar(name string) error {

	f, err := NewFtar(name, defaultGroup)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	/*
		if len(c.ftars) > MaxFtars {
			c.clear()
		}
	*/
	c.ftars[f.Id] = f
	return nil
}

// GetList returns the actual list of flexatars (keys == images) that can be used for animating
func (c *Cache) GetList(g string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var l []string
	for k, f := range c.ftars {
		if (len(g) == 0) || (g == f.Group) {
			l = append(l, k)
		}
	}
	return json.Marshal(l)
}

// GetIcon() returns the real path to the image
func (c *Cache) GetIcon(name string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	f, ok := c.ftars[name]
	if !ok {
		return "", fmt.Errorf("no such file")
	}
	return f.Icon, nil
}

// ResolveFtar() returns the path to the ftar file
func (c *Cache) ResolveFtar(name string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, ok := c.ftars[name]
	if !ok {
		return "", fmt.Errorf("no such file")
	}
	return f.File, nil
}

// AddFtar() > LOCK
func (c *Cache) clear() {
	var oldest *Ftar
	oldest = nil
	for _, f := range c.ftars {
		if f.Fixed {
			continue
		}
		if oldest == nil {
			oldest = f
			continue
		}
		if oldest.Date.After(f.Date) {
			oldest = f
		}
	}
	if oldest != nil {
		delete(c.ftars, oldest.Id)
		os.Remove(oldest.File)
		os.Remove(oldest.Icon)
	}
}
