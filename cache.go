package validator

import "sync"

type fieldCache struct {
	backend sync.Map
}

func (c *fieldCache) Get(path string) (fc []*fieldContext, has bool) {
	val, has := c.backend.Load(path)
	if has {
		return val.([]*fieldContext), true
	}
	return nil, false
}

func (c *fieldCache) Store(path string, fc []*fieldContext) {
	c.backend.Store(path, fc)
}
