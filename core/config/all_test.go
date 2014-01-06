//  ---------------------------------------------------------------------------
//
//  all_test.go
//
//  Written by Jared Chavez (2014-01-01)
//  Owned by Jared Chavez <xaevman@gmail.com>
//
//  Copyright (c) 2014 Jared Chavez
//
//  -----------

package config

import(
	"log"
	"testing"
)

func TestConfig(t *testing.T) {
	InitEnvProvider(1)
	InitIniProvider("./test.ini", 2)

	key         := "PATH"
	data, entry := GetAllVals(key, "/bin:/sbin")
	printConfig(key, data, entry.Parser())

	key         = "Ini.Section.key1"
	data, entry = GetAllVals(key, "default1")
	printConfig(key, data, entry.Parser())

	key         = "This.Key.Shouldnt.exist"
	data, entry = GetAllVals(key, "default3")
	printConfig(key, data, entry.Parser())
}

func printConfig(key string, vals []string, parser ConfigProvider) {
	if vals == nil {
		return
	}

	for i, v := range vals {
		log.Printf("%v.%v[%v]: %v", parser.Name(), key, i, v)
	}
}
