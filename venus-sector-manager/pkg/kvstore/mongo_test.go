package kvstore_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/strikesecurity/strikememongo"

	"github.com/ipfs-force-community/venus-cluster/venus-sector-manager/pkg/kvstore"
)

var (
	mongoServer   *strikememongo.Server
	testKey1      = []byte("testKey1")
	testKey2      = []byte("testKey2")
	testKey3      = []byte("testKey3")
	testPrefixKey = []byte("testKey")

	testValue1 = []byte("testValue1")
	testValue2 = []byte("testValue2")
	testValue3 = []byte("testValue3")
)

func TestMain(m *testing.M) {
	var err error
	mongoServer, err = strikememongo.Start("4.0.5")
	if err != nil {
		log.Fatal(err)
	}
	defer mongoServer.Stop()

	os.Exit(m.Run())
}
func TestMongoStore_PutGet(t *testing.T) {
	kv, err := kvstore.OpenMongo(context.TODO(), mongoServer.URI(), "vcs", "test")
	require.NoError(t, err)
	ctx := context.TODO()

	err = kv.Put(ctx, testKey1, testValue1)
	require.NoError(t, err)

	val, err := kv.Get(ctx, testKey1)
	require.NoError(t, err)
	require.Equal(t, testValue1, val)

	_, err = kv.Get(ctx, testKey2)
	require.Equal(t, kvstore.ErrKeyNotFound, err)

	err = kv.Put(ctx, testKey1, testValue2)
	require.NoError(t, err)

	val, err = kv.Get(ctx, testKey1)
	require.NoError(t, err)
	require.Equal(t, testValue2, val)
}

func TestMongoStore_Has(t *testing.T) {
	kv, err := kvstore.OpenMongo(context.TODO(), mongoServer.URI(), "vcs", "test")
	require.NoError(t, err)
	ctx := context.TODO()

	err = kv.Put(ctx, testKey1, testValue1)
	require.NoError(t, err)

	exist, err := kv.Has(ctx, testKey1)
	require.NoError(t, err)
	require.Equal(t, true, exist)

	exist, err = kv.Has(ctx, testKey2)
	require.NoError(t, err)
	require.Equal(t, false, exist)
}

// this case will also test the usage of iter
func TestMongoStore_Scan(t *testing.T) {
	kv, err := kvstore.OpenMongo(context.TODO(), mongoServer.URI(), "vcs", "test")
	require.NoError(t, err)
	ctx := context.TODO()

	err = kv.Put(ctx, testKey1, testValue1)
	require.NoError(t, err)
	err = kv.Put(ctx, testKey2, testValue2)
	require.NoError(t, err)
	err = kv.Put(ctx, testKey3, testValue3)
	require.NoError(t, err)

	iter, err := kv.Scan(ctx, testPrefixKey)
	require.NoError(t, err)

	cnt := 0
	for iter.Next() {
		cnt++
		v := kvstore.Val{}
		err = iter.View(ctx, func(val kvstore.Val) error {
			v = val
			return nil
		})
		require.NoError(t, err)
		switch {
		case bytes.Equal(iter.Key(), testKey1):
			require.Equal(t, testValue1, v)
		case bytes.Equal(iter.Key(), testKey2):
			require.Equal(t, testValue2, v)
		case bytes.Equal(iter.Key(), testKey3):
			require.Equal(t, testValue3, v)
		default:
			require.Error(t, fmt.Errorf("failed to match iter.Key"))
		}
	}
	require.Equal(t, 3, cnt)
}

func TestMongoStore_ScanNil(t *testing.T) {
	kv, err := kvstore.OpenMongo(context.TODO(), mongoServer.URI(), "vcs", "test")
	require.NoError(t, err)
	ctx := context.TODO()

	err = kv.Put(ctx, testKey1, testValue1)
	require.NoError(t, err)
	err = kv.Put(ctx, testKey2, testValue2)
	require.NoError(t, err)
	err = kv.Put(ctx, testKey3, testValue3)
	require.NoError(t, err)

	err = kv.Put(ctx, []byte("tmp"), testValue3)
	require.NoError(t, err)
	// should scan all key
	iter, err := kv.Scan(ctx, nil)
	require.NoError(t, err)

	cnt := 0
	for iter.Next() {
		cnt++
	}
	require.Equal(t, 4, cnt)
}
