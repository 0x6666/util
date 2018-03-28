package cache

import (
	"errors"
	"time"
)

var _cache Cache

// Length of time to cache an item.
const (
	DefaultExpiryTime  = time.Duration(0)
	ForEverNeverExpiry = time.Duration(-1)
)

var (
	ErrCacheMiss    = errors.New("cache: key not found")
	ErrNotStored    = errors.New("cache: not stored")
	ErrInvalidValue = errors.New("cache: invalid value")
	ErrInited       = errors.New("cache: inited")
)

type Cache interface {
	// Get the content associated with the given key. decoding it into the given
	// pointer.
	//
	// Returns:
	//   - nil if the value was successfully retrieved and ptrValue set
	//   - ErrCacheMiss if the value was not in the cache
	//   - an implementation specific error otherwise
	Get(key string, ptrValue interface{}) error

	// Set the given key/value in the cache, overwriting any existing value
	// associated with that key.  Keys may be at most 250 bytes in length.
	//
	// Returns:
	//   - nil on success
	//   - an implementation specific error otherwise
	Set(key string, value interface{}, expires time.Duration) error

	// Delete the given key from the cache.
	//
	// Returns:
	//   - nil on a successful delete
	//   - ErrCacheMiss if the value was not in the cache
	//   - an implementation specific error otherwise
	Delete(key string) error

	// Increment the value stored at the given key by the given amount.
	// The value silently wraps around upon exceeding the uint64 range.
	//
	// Returns the new counter value if the operation was successful, or:
	//   - ErrCacheMiss if the key was not found in the cache
	//   - an implementation specific error otherwise
	Increment(key string, n uint64) (newValue uint64, err error)

	// Decrement the value stored at the given key by the given amount.
	// The value is capped at 0 on underflow, with no error returned.
	//
	// Returns the new counter value if the operation was successful, or:
	//   - ErrCacheMiss if the key was not found in the cache
	//   - an implementation specific error otherwise
	Decrement(key string, n uint64) (newValue uint64, err error)

	// Expire all cache entries immediately.
	// This is not implemented for the memcached cache (intentionally).
	// Returns an implementation specific error if the operation failed.
	ClearAll() error
}

func Get(key string, ptrValue interface{}) error                  { return _cache.Get(key, ptrValue) }
func Delete(key string) error                                     { return _cache.Delete(key) }
func Increment(key string, n uint64) (newValue uint64, err error) { return _cache.Increment(key, n) }
func Decrement(key string, n uint64) (newValue uint64, err error) { return _cache.Decrement(key, n) }
func ClearAll() error                                             { return _cache.ClearAll() }
func Set(key string, value interface{}, expires time.Duration) error {
	return _cache.Set(key, value, expires)
}

func InitRedisCache(host string, password string, dbNum int, defaultExpiration time.Duration) error {
	if _cache != nil {
		return ErrInited
	}
	_cache = newRedisCache(host, password, dbNum, defaultExpiration)
	return nil
}
