package cache

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisCache wraps the Redis client to meet the Cache interface.
type RedisCache struct {
	p                 *redis.Pool
	defaultExpiration time.Duration
}

// NewRedisCache returns a new RedisCache with given parameters
// until redigo supports sharding/clustering, only one host will be in hostList
func newRedisCache(host string, password string, dbNum int, defaultExpiration time.Duration) RedisCache {
	var pool = &redis.Pool{
		MaxIdle:     5,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host,
				redis.DialConnectTimeout(time.Millisecond*10000),
				redis.DialReadTimeout(time.Millisecond*5000),
				redis.DialWriteTimeout(time.Millisecond*5000))
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err = c.Do("AUTH", password); err != nil {
					_ = c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err = c.Do("PING"); err != nil {
					_ = c.Close()
					return nil, err
				}
			}

			_, err = c.Do("SELECT", dbNum)
			if err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return RedisCache{pool, defaultExpiration}
}

func (c RedisCache) Set(key string, value interface{}, expires time.Duration) error {
	conn := c.p.Get()
	defer conn.Close()
	return c.invoke(conn.Do, key, value, expires)
}

func (c RedisCache) Get(key string, ptrValue interface{}) error {
	conn := c.p.Get()
	defer conn.Close()
	raw, err := conn.Do("GET", key)
	if err != nil {
		return err
	} else if raw == nil {
		return ErrCacheMiss
	}
	item, err := redis.Bytes(raw, err)
	if err != nil {
		return err
	}
	return Deserialize(item, ptrValue)
}

func exists(conn redis.Conn, key string) (bool, error) {
	return redis.Bool(conn.Do("EXISTS", key))
}

func (c RedisCache) Delete(key string) error {
	conn := c.p.Get()
	defer conn.Close()
	existed, err := redis.Bool(conn.Do("DEL", key))
	if err == nil && !existed {
		err = ErrCacheMiss
	}
	return err
}

func (c RedisCache) Increment(key string, delta uint64) (uint64, error) {
	conn := c.p.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that. Since we need to do increment
	// ourselves instead of natively via INCRBY (redis doesn't support wrapping), we get the value
	// and do the exists check this way to minimize calls to Redis
	val, err := conn.Do("GET", key)
	if err != nil {
		return 0, err
	} else if val == nil {
		return 0, ErrCacheMiss
	}
	currentVal, err := redis.Int64(val, nil)
	if err != nil {
		return 0, err
	}
	sum := currentVal + int64(delta)
	_, err = conn.Do("SET", key, sum)
	if err != nil {
		return 0, err
	}
	return uint64(sum), nil
}

func (c RedisCache) Decrement(key string, delta uint64) (newValue uint64, err error) {
	conn := c.p.Get()
	defer conn.Close()
	// Check for existance *before* increment as per the cache contract.
	// redis will auto create the key, and we don't want that, hence the exists call
	existed, err := exists(conn, key)
	if err != nil {
		return 0, err
	} else if !existed {
		return 0, ErrCacheMiss
	}
	// Decrement contract says you can only go to 0
	// so we go fetch the value and if the delta is greater than the amount,
	// 0 out the value
	currentVal, err := redis.Int64(conn.Do("GET", key))
	if err != nil {
		return 0, err
	}
	if delta > uint64(currentVal) {
		var tempint int64
		tempint, err = redis.Int64(conn.Do("DECRBY", key, currentVal))
		return uint64(tempint), err
	}
	tempint, err := redis.Int64(conn.Do("DECRBY", key, delta))
	return uint64(tempint), err
}

func (c RedisCache) ClearAll() error {
	conn := c.p.Get()
	defer conn.Close()
	_, err := conn.Do( /*"FLUSHALL"*/ "FLUSHDB")
	return err
}

func (c RedisCache) invoke(f func(string, ...interface{}) (interface{}, error),
	key string, value interface{}, expires time.Duration) error {

	switch expires {
	case DefaultExpiryTime:
		expires = c.defaultExpiration
	case ForEverNeverExpiry:
		expires = time.Duration(0)
	}

	b, err := Serialize(value)
	if err != nil {
		return err
	}
	conn := c.p.Get()
	defer conn.Close()

	if expires > 0 {
		_, err = f("SETEX", key, int32(expires/time.Second), b)
		return err
	}
	_, err = f("SET", key, b)
	return err
}
