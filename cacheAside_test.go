package cacheAside

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
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
