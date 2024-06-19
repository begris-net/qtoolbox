/*
* Copyright (c) 2024 Bjoern Beier.
*
* Permission is hereby granted, free of charge, to any person obtaining a copy of
* this software and associated documentation files (the "Software"), to deal in
* the Software without restriction, including without limitation the rights to
* use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
* the Software, and to permit persons to whom the Software is furnished to do so,
* subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
* FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
* COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
* IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
* CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package cache

import (
	"encoding/json"
	"fmt"
	"github.com/begris-net/qtoolbox/internal/candidate"
	"github.com/begris-net/qtoolbox/internal/log"
	"github.com/begris-net/qtoolbox/internal/util"
	"os"
	"path"
	"time"
)

type Cache[T any] struct {
	cachePath    string
	cacheTtl     time.Duration
	releaseCache releaseCache[T]
}

type releaseCache[T any] struct {
	UpdateTime time.Time `json:"update_time"`
	Provider   string    `json:"provider"`
	Releases   T         `json:"releases"`
}

func (c *Cache[T]) SetTTL(ttl *time.Duration) {
	c.cacheTtl = util.SafeDeref(ttl)
}

func (c *Cache[T]) SetCachePath(cachePath *string) {
	c.cachePath = util.SafeDeref(cachePath)
}

func (c *Cache[T]) ensuredCachePathExists(cachePath string) {
	err := os.MkdirAll(cachePath, 0750)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Error creating cache path %s.", cachePath), log.Logger.Args("err", err))
	}
}

func (c *Cache[T]) GetCachedReleases(provider candidate.CandidateProvider, refreshFunction func() T) T {
	cachePath := path.Join(c.cachePath, provider.Product, provider.ProviderRepoId+".cache")
	c.ensuredCachePathExists(path.Dir(cachePath))
	log.Logger.Info(fmt.Sprintf("Looking up release cache for provider %s", provider.ProviderRepoId))
	cacheFile, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(cacheFile).Decode(&c.releaseCache)
	var refresh bool
	if err == nil {
		refresh = time.Now().After(c.releaseCache.UpdateTime.Add(c.cacheTtl))
		log.Logger.Debug(fmt.Sprintf("Cache for provider %s is valid until %s - refresh = %v.", provider.ProviderRepoId,
			c.releaseCache.UpdateTime.Add(c.cacheTtl).Format(time.RFC1123), refresh))
		if refresh {
			log.Logger.Info(fmt.Sprintf("Cache for provider %s is outdated. Last updated at %s.", provider.ProviderRepoId,
				c.releaseCache.UpdateTime.Format(time.RFC1123)))
		} else {
			log.Logger.Info(fmt.Sprintf("Cache for provider %s is valid until %s. Last updated at %s.", provider.ProviderRepoId,
				c.releaseCache.UpdateTime.Add(c.cacheTtl).Format(time.RFC1123), c.releaseCache.UpdateTime.Format(time.RFC1123)))
		}
	} else {
		refresh = true
		log.Logger.Warn(fmt.Sprintf("Cache for provider %s is invaild. Forcing refresh.", provider.ProviderRepoId))
	}

	if refresh {

		releases := refreshFunction()

		c.releaseCache = releaseCache[T]{
			UpdateTime: time.Now(),
			Provider:   provider.ProviderRepoId,
			Releases:   releases,
		}

		_, err := cacheFile.Seek(0, 0)
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Error while updating release cache for %s/%s. Could not reset file offset.",
				provider.Product, provider.ProviderRepoId), log.Logger.Args("err", err))
		}

		err = json.NewEncoder(cacheFile).Encode(c.releaseCache)
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Error while updating release cache for %s/%s", provider.Product, provider.ProviderRepoId),
				log.Logger.Args("err", err))
		}
		cacheFile.Sync()
	}
	cacheFile.Close()

	return c.releaseCache.Releases
}
