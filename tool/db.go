package tool

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
)

const (
	ROOT_BKT    = "m3u8"
	URL_BKT     = "url"
	FAILED_BKT  = "failedUrl"
	SUCCESS_BKT = "successUrl"
)

type UrlDB struct {
	db *bolt.DB
}

func OpenUrlDB(path string) (*UrlDB, error) {
	options := *bolt.DefaultOptions
	db, err := bolt.Open(path, 0644, &options)
	if err != nil {
		return nil, err
	}

	return &UrlDB{
		db: db,
	}, nil
}

func (self *UrlDB) update(tx func(tx *bolt.Tx) error) error {
	return self.db.Update(tx)
}

func (self *UrlDB) view(tx func(tx *bolt.Tx) error) error {
	return self.db.View(tx)
}

func (self *UrlDB) getBucket(tx *bolt.Tx, keys ...[]byte) *bolt.Bucket {
	bkt := tx.Bucket(keys[0])

	for _, key := range keys[1:] {
		if bkt == nil {
			break
		}
		bkt = bkt.Bucket(key)
	}

	return bkt
}

func (self *UrlDB) createBucketIfNotExists(tx *bolt.Tx, keys ...[]byte) (*bolt.Bucket, error) {
	bkt, err := tx.CreateBucketIfNotExists(keys[0])
	if err != nil {
		return nil, err
	}

	for _, key := range keys[1:] {
		bkt, err = bkt.CreateBucketIfNotExists(key)
		if err != nil {
			return nil, err
		}
	}

	return bkt, nil
}

func (self *UrlDB) bucketPut(bkt *bolt.Bucket, key string, value string) error {
	if bkt == nil {
		return fmt.Errorf("bucket is null")
	}

	return bkt.Put([]byte(key), []byte(value))
}

func (self *UrlDB) getRootBucketName() []byte {
	return []byte(ROOT_BKT)
}

func (self *UrlDB) getUrlBucket(tx *bolt.Tx) *bolt.Bucket {
	return self.getBucket(tx, self.getRootBucketName(), []byte(URL_BKT))
}

func (self *UrlDB) createUrlBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	return self.createBucketIfNotExists(tx, self.getRootBucketName(), []byte(URL_BKT))
}

func (self *UrlDB) add(getBucket func(tx *bolt.Tx) *bolt.Bucket,
	createBucket func(tx *bolt.Tx) (*bolt.Bucket, error),
	putBucket func(bkt *bolt.Bucket, key string, value string) error,
	key string,
	value string) error {
	return self.update(func(tx *bolt.Tx) error {
		bkt := getBucket(tx)
		if bkt == nil {
			b, err := createBucket(tx)
			if err != nil {
				return err
			}
			bkt = b
		}
		return putBucket(bkt, key, value)
	})
}

func (self *UrlDB) delete(getBucket func(tx *bolt.Tx) *bolt.Bucket,
	key string) error {
	return self.update(func(tx *bolt.Tx) error {
		bkt := getBucket(tx)
		if bkt == nil {
			return nil
		}

		err := bkt.Delete([]byte(key))
		if err != nil {
			if err == bolt.ErrBucketNotFound {
				err = fmt.Errorf("key %v: %w", key, err)
			}
		}
		return err
	})
}

func (self *UrlDB) list(getBucket func(tx *bolt.Tx) *bolt.Bucket) ([]string, error) {
	result := make([]string, 0)
	return result, self.view(func(tx *bolt.Tx) error {
		bkt := getBucket(tx)
		if bkt == nil {
			return fmt.Errorf("find bucket failed")
		}

		return bkt.ForEach(func(k, v []byte) error {
			result = append(result, string(k))
			return nil
		})
	})
}

func (self *UrlDB) has(list func() ([]string, error), url string) (bool, error) {
	urls, err := list()
	if err != nil {
		return false, err
	}
	for _, i := range urls {
		if i == url {
			return true, nil
		}
	}
	return false, nil
}

func (self *UrlDB) getFailedUrlBucket(tx *bolt.Tx) *bolt.Bucket {
	return self.getBucket(tx, self.getRootBucketName(), []byte(FAILED_BKT))
}

func (self *UrlDB) createFailedUrlBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	return self.createBucketIfNotExists(tx, self.getRootBucketName(), []byte(FAILED_BKT))
}

func (self *UrlDB) AddUrl(url string) error {
	return self.add(self.getUrlBucket, self.createUrlBucket, self.bucketPut, url, "0")
}

func (self *UrlDB) DeleteUrl(url string) error {
	return self.delete(self.getUrlBucket, url)
}

func (self *UrlDB) ListUrls() ([]string, error) {
	return self.list(self.getUrlBucket)
}

func (self *UrlDB) IsHaveUrl(url string) (bool, error) {
	return self.has(self.ListUrls, url)
}

func (self *UrlDB) AddFailedUrl(url string) error {
	return self.add(self.getFailedUrlBucket, self.createFailedUrlBucket, self.bucketPut, url, "0")
}

func (self *UrlDB) getSuccessUrlBucket(tx *bolt.Tx) *bolt.Bucket {
	return self.getBucket(tx, self.getRootBucketName(), []byte(SUCCESS_BKT))
}

func (self *UrlDB) createSuccessUrlBucket(tx *bolt.Tx) (*bolt.Bucket, error) {
	return self.createBucketIfNotExists(tx, self.getRootBucketName(), []byte(SUCCESS_BKT))
}

func (self *UrlDB) AddSuccessUrl(url string) error {
	return self.add(self.getSuccessUrlBucket, self.createSuccessUrlBucket, self.bucketPut, url, "0")
}

func (self *UrlDB) IsFailedUrl(url string) (bool, error) {
	return self.has(self.ListFailedUrls, url)
}

func (self *UrlDB) ListFailedUrls() ([]string, error) {
	return self.list(self.getFailedUrlBucket)
}

func (self *UrlDB) DeleteFailedUrl(url string) error {
	return self.delete(self.getFailedUrlBucket, url)
}

func (self *UrlDB) ListSuccessUrls() ([]string, error) {
	return self.list(self.getSuccessUrlBucket)
}

func (self *UrlDB) DeleteSuccessUrl(url string) error {
	return self.delete(self.getSuccessUrlBucket, url)
}

func (self *UrlDB) IsSuccessUrl(url string) (bool, error) {
	return self.has(self.ListSuccessUrls, url)
}

func (self *UrlDB) Close() error {
	return self.db.Close()
}
