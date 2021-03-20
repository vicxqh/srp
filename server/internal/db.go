package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/vicxqh/srp/log"
	"github.com/vicxqh/srp/types"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExist = errors.New("already existed")
)

var db *bolt.DB

var (
	BucketServiceMeta = []byte("service")
)

type ServiceMeta struct {
	ID          string // unique
	Addr        string
	Description string
}

func InitDB() {
	var err error
	db, err = bolt.Open("service.db", 0600, nil)
	if err != nil {
		log.Fatal("failed to open db, %v", err)
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists(BucketServiceMeta)
		if err != nil {
			log.Fatal("failed to create bucket, %v", err)
		}
		return err
	})
}

func CloseDB() {
	db.Close()
}

func listServices(ctx context.Context) ([]types.Service, error) {
	var services []types.Service
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketServiceMeta)
		return bucket.ForEach(func(k, v []byte) error {
			s, err := composeService(v)
			if err != nil {
				log.Info("failed to compose service, %s, error %v", string(v), err)
				return err
			}
			services = append(services, s)
			return nil
		})
	})
	return services, err
}

func composeService(metadata []byte) (types.Service, error) {
	var s types.Service
	var meta ServiceMeta
	err := json.Unmarshal(metadata, &meta)
	if err != nil {

		return s, nil
	}

	s.ID = meta.ID
	s.Addr = meta.Addr
	s.Description = meta.Description

	exp := GetExposure(s.ID)
	if exp != nil {
		s.ExposedBy = exp.AgentId
		s.ServerPort = exp.Port
	}

	return s, nil
}

func getService(ctx context.Context, id string) (types.Service, error) {
	var service types.Service
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketServiceMeta)
		meta := bucket.Get([]byte(id))
		if meta == nil {
			log.Warn("service %s not exist", id)
			return ErrNotFound
		}
		var err error
		service, err = composeService(meta)
		return err
	})
	return service, err
}

func updateService(ctx context.Context, id string, svc types.Service) error {
	if svc.ID == "" {
		return errors.New("Service.ID can NOT be empty")
	}
	if id != svc.ID {
		return fmt.Errorf("Service.ID(%s) didn't match requested id(%s), svc.ID, id")
	}
	meta := ServiceMeta{
		ID:          svc.ID,
		Addr:        svc.Addr,
		Description: svc.Description,
	}
	metadata, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketServiceMeta)
		return bucket.Put([]byte(id), metadata)
	})
}

func createService(ctx context.Context, svc types.Service) error {
	if svc.ID == "" {
		return errors.New("Service.ID can NOT be empty")
	}
	meta := ServiceMeta{
		ID:          svc.ID,
		Addr:        svc.Addr,
		Description: svc.Description,
	}
	metadata, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketServiceMeta)
		old := bucket.Get([]byte(svc.ID))
		if old != nil {
			return ErrAlreadyExist
		}
		return bucket.Put([]byte(svc.ID), metadata)
	})
}

func deleteService(ctx context.Context, id string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BucketServiceMeta)
		return bucket.Delete([]byte(id))
	})
}
