package cacheAside

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	dataBase = map[string]string{
		"hello": "world",
		"foo":   "bar",
	}
)

func init() {
	for i := 0; i < 3000; i++ {
		dataBase[strconv.Itoa(i)] = strconv.Itoa(i)
	}
}

var (
	dbNotFound = errors.New("not found")
)

func TestOption(t *testing.T) {
	Init(&Option{
		BatchSize:          100,
		DefaultCacheExpire: time.Hour * 24 * 3,
		MissCacheExpire:    time.Minute * 2,
		CleanInterval:      time.Hour * 2,
	})
	if batchSize != 100 {
		t.Error("batchSize not equal")
	}
	if defaultCacheExpire != time.Hour*24*3 {
		t.Error("defaultCacheExpire not equal")
	}
	if defaultMissCacheExpire != time.Minute*2 {
		t.Error("defaultMissCacheExpire not equal")
	}
	if defaultCleanInterval != time.Hour*2 {
		t.Error("defaultCleanInterval not equal")
	}
}

func find(key string) (string, error) {
	if v, ok := dataBase[key]; ok {
		return v, nil
	}
	return "", dbNotFound
}

func findMany(keys []string) (map[string]string, error) {
	res := make(map[string]string)
	for _, key := range keys {
		if v, ok := dataBase[key]; ok {
			res[key] = v
		}
	}
	return res, nil
}

func found(id string) (string, bool, error) {
	v, err := find("hello")
	if err != nil {
		return "", false, err
	}
	if err == dbNotFound {
		return "", true, nil
	}
	return v, false, nil
}

func notFound(id string) (string, bool, error) {
	return "", true, nil
}

func foundError(id string) (string, bool, error) {
	return "", true, errors.New("error")
}

func TestGetWithFound(t *testing.T) {
	debugInit(nil)
	res, err := Get("key", found)
	if err != nil {
		t.Error(err)
	}
	if res != "world" {
		t.Error("res not hello")
	}
	res, err = Get("key", found)
	if err != nil {
		t.Error(err)
	}
	if res != "world" {
		t.Error("res not hello")
	}
}

type User struct {
	Id   string
	Name string
	Age  int
}

func findUser(id string) (*User, bool, error) {
	return &User{
		Id:   "1",
		Name: "tom",
		Age:  18,
	}, false, nil
}

func GetUserInfo(id string) (*User, error) {
	s, err := Get(id, findUser)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func TestGetUserInfo(t *testing.T) {
	debugInit(nil)
	u, err := GetUserInfo("1")
	if err != nil {
		t.Error(err)
	}
	if u == nil {
		t.Error("u is nil")
	}
	t.Log(u)

	_, err = GetUserInfo("1")
	if err != nil {
		t.Error(err)
	}
	t.Log(u)
}

func TestGetWithNotFound(t *testing.T) {
	debugInit(nil)
	_, err := Get("not exist", notFound)
	if err != ErrNotFound {
		t.Error(err)
	}

	_, err = Get("not exist", notFound)
	if err != ErrNotFound {
		t.Error(err)
	}
}

func TestGetWithError(t *testing.T) {
	debugInit(nil)
	_, err := Get("hello", foundError)
	if err == nil {
		t.Error(err)
	}

	_, err = Get("hello", foundError)
	if err == nil {
		t.Error(err)
	}

}

func TestGetThenDel(t *testing.T) {
	debugInit(nil)
	_, err := Get("hello", found)
	if err != nil {
		t.Error(err)
	}
	Delete("hello")
	exist := debugExist("hello")
	if exist {
		t.Error("hello should not exist")
	}

}

func foundMany(id []string) (map[string]string, error) {
	return findMany([]string{"hello", "foo"})
}

func notFoundMany(id []string) (map[string]string, error) {
	return nil, nil
}

func foundManyError(id []string) (map[string]string, error) {
	return nil, errors.New("error")
}

func TestGetsWithFoundMany(t *testing.T) {
	debugInit(nil)
	mp, err := MultiGet([]string{"hello", "foo"}, foundMany)
	if err != nil {
		t.Error(err)
	}
	if mp["hello"] != "world" {
		t.Error("res not hello")
	}
	if mp["foo"] != "bar" {
		t.Error("res not foo")
	}
	mp, err = MultiGet([]string{"hello", "foo"}, foundMany)
	if err != nil {
		t.Error(err)
	}
	if mp["hello"] != "world" {
		t.Error("res not hello")
	}
	if mp["foo"] != "bar" {
		t.Error("res not foo")
	}
}

func TestGetsWithNotFoundMany(t *testing.T) {
	debugInit(nil)
	mp, err := MultiGet([]string{"hello", "foo"}, notFoundMany)
	if err != nil {
		t.Error(err)
	}
	if mp["hello"] != "" {
		t.Error("res not nil")
	}
	if mp["foo"] != "" {
		t.Error("res not nil")
	}

	mp, err = MultiGet([]string{"hello", "foo"}, notFoundMany)
	if err != nil {
		t.Error(err)
	}
	if mp["hello"] != "" {
		t.Error("res not nil")
	}
	if mp["foo"] != "" {
		t.Error("res not nil")
	}

}

func TestGetsWithfoundManyError(t *testing.T) {
	debugInit(nil)
	_, err := MultiGet([]string{"hello", "foo"}, foundManyError)
	if err == nil {
		t.Error(err)
	}

	_, err = MultiGet([]string{"hello", "foo"}, foundManyError)
	if err == nil {
		t.Error(err)
	}

}

func TestMultiGet(t *testing.T) {
	debugInit(nil)
	keys := make([]string, 0)
	for i := 0; i < 2000; i++ {
		keys = append(keys, fmt.Sprintf("%d", i))
	}
	mp, err := MultiGet(keys, findMany)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 2000; i++ {
		if mp[fmt.Sprintf("%d", i)] != fmt.Sprintf("%d", i) {
			t.Error("res not v")
		}
	}
	mp, err = MultiGet(keys, findMany)
	if err != nil {
		t.Error(err)
	}
	Delete(keys...)
	for i := 0; i < 2000; i++ {
		if debugExist(fmt.Sprintf("%d", i)) {
			t.Error("still in")
		}
	}
}

func TestMultiGetWithMissCache(t *testing.T) {
	debugInit(nil)
	keys := make([]string, 0)
	for i := 0; i < 4000; i++ {
		keys = append(keys, fmt.Sprintf("%d", i))
	}
	mp, err := MultiGet(keys, findMany)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 3000; i++ {
		if mp[fmt.Sprintf("%d", i)] != fmt.Sprintf("%d", i) {
			t.Error("res not v")
		}
	}
	mp, err = MultiGet(keys, findMany)
	if err != nil {
		t.Error(err)
	}
	Delete(keys...)
	for i := 0; i < 3000; i++ {
		if debugExist(fmt.Sprintf("%d", i)) {
			t.Error("still in")
		}
	}
	for i := 3000; i < 4000; i++ {
		if mp[fmt.Sprintf("%d", i)] != "" {
			t.Error("it should be empty")
		}
	}
}

func TestMultiGetCacheMiss(t *testing.T) {
	debugInit(nil)
	mp, err := MultiGet([]string{"not exist"}, findMany)
	if err != nil {
		t.Error(err)
	}
	if mp["not exist"] != "" {
		t.Error("should be empty")
	}
	if !debugExist("not exist") {
		t.Error("should  exist")
	}
	mp, err = MultiGet([]string{"not exist"}, findMany)
	if err != nil {
		t.Error(err)
	}
	if mp["not exist"] != "" {
		t.Error("should be empty")
	}
}

func TestGetSingleFlight(t *testing.T) {
	debugInit(nil)
	accessDb := atomic.Int32{}
	foundWithCounter := func(id string) (string, bool, error) {
		accessDb.Add(1)
		return found(id)
	}
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			_, err := Get("hello", foundWithCounter)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	t.Log(accessDb.Load())
	if accessDb.Load() >= 1000 {
		t.Error("should much less than 1000")
	}
}

func TestCacheExpire(t *testing.T) {
	debugInit(&Option{
		BatchSize:          100,
		DefaultCacheExpire: 5 * time.Second,
		MissCacheExpire:    time.Minute,
		CleanInterval:      time.Second * 3,
	})
	_, err := Get("hello", found)
	if err != nil {
		t.Fatal(err)
	}
	if !debugExist("hello") {
		t.Error("should exist")
	}
	time.Sleep(time.Second * 6)
	if debugExist("hello") {
		t.Fatal("should not exist")
	}
	_, err = Get("hello", found)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkGet(b *testing.B) {
	debugInit(nil)
	for i := 0; i < b.N; i++ {
		_, err := Get("hello", found)
		if err != nil {
			b.Fatal(err)
		}
	}
}
