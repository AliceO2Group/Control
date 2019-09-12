/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package configuration

import (
	"github.com/hashicorp/consul/api"
	"errors"
	"strconv"
	"strings"
	"gopkg.in/yaml.v2"
)

type ConsulSource struct {
	uri string
	kv  *api.KV
}

func newConsulSource(uri string) (cc *ConsulSource, err error) {
	cfg := api.DefaultConfig()
	cfg.Address = uri
	cli, err := api.NewClient(cfg)
	if err != nil {
		return
	}
	cc = &ConsulSource{
		uri: uri,
		kv: cli.KV(),
	}
	return
}

func NewConsulSource(uri string) (cc *ConsulSource, err error) {
	cfg := api.DefaultConfig()
	cfg.Address = uri
	cli, err := api.NewClient(cfg)
	if err != nil {
		return
	}
	cc = &ConsulSource{
		uri: uri,
		kv: cli.KV(),
	}
	return
}

func (cc *ConsulSource) GetNextUInt32(key string) (value uint32, err error) {
	kvp, _, err := cc.kv.Get(formatKey(key), &api.QueryOptions{RequireConsistent: true})
	if err != nil {
		return
	}
	var value64 uint64
	value64, err = strconv.ParseUint(string(kvp.Value[:]), 10, 32)
	if err != nil {
		return
	}
	value = uint32(value64)
	value++
	kvp.Value = []byte(strconv.FormatUint(uint64(value), 10))
	var ok bool
	ok, _, err = cc.kv.CAS(kvp, nil) // Check-And-Set call, relies on ModifyIndex in KVPair
	if err != nil {
		return
	}
	if !ok {
		err = errors.New("cannot write back incremented CAS key")
	}
	return
}

func (cc *ConsulSource) Get(key string) (value string, err error) {
	kvp, _, err := cc.kv.Get(formatKey(key), nil)
	if err != nil {
		return
	}
	if kvp != nil {
		value = string(kvp.Value[:])
	} else {
		value = ""
	}
	return
}

func (cc *ConsulSource) GetKeysByPrefix(keyPrefix string, separator string)(value []string, err error) {
	keyPrefix = formatKey(keyPrefix)
	keys, _, err := cc.kv.Keys(keyPrefix, separator, nil)
	if err != nil {
		return
	}
	return keys, nil
}

func (cc *ConsulSource) GetRecursive(key string) (value Item, err error) {
	requestKey := formatKey(key)
	kvps, _, err := cc.kv.List(requestKey, nil)
	if err != nil {
		return
	}
	for _, kvp := range kvps {
		kvp.Key = stripRequestKey(requestKey, kvp.Key)
	}
	return mapify(kvps), nil
}

func (cc *ConsulSource) GetRecursiveYaml(key string) (value []byte, err error) {
	var item Item
	item, err = cc.GetRecursive(key)
	if err != nil {
		return
	}
	value, err = yaml.Marshal(item)
	return
}

func (cc *ConsulSource) Put(key string, value string) (err error) {
	kvp := &api.KVPair{Key: formatKey(key), Value: []byte(value)}
	_, err = cc.kv.Put(kvp, nil)
	return
}

func (cc *ConsulSource) PutRecursive(string, Item) error {
	// FIXME
	panic("implement me")
}

func (cc *ConsulSource) PutRecursiveYaml(string, []byte) error {
	// FIXME
	panic("implement me")
}

func (cc *ConsulSource) Exists(key string) (exists bool, err error) {
	kvp, _, err := cc.kv.Get(formatKey(key), nil)
	if err != nil {
		return
	}
	exists = kvp != nil
	return
}

func formatKey(key string) (consulKey string) {
	// Trim leading slashes
	consulKey = strings.TrimLeft(key, "/")
	return
}

func stripRequestKey(requestKey string, responseKey string) string {
	// The request key is prefixed to the response keys, this strips that from it.
	return strings.TrimPrefix(responseKey, requestKey)
}

func mapify(kvps api.KVPairs) Map {
	// Our output Map (=map[string]Item)
	m := make(Map)

	// We accumulate a partial map here, by stripping the leftmost
	// part of the key as prefix, and associating it with a slice
	// of KVPairs for that prefix.
	prefixSet := make(map[string]api.KVPairs)

	for _, kvp := range kvps {
		if len(strings.TrimSpace(kvp.Key)) == 0 {
			continue
		}

		i := strings.IndexByte(kvp.Key, '/')
		if i == 0 {
			// Looks like the key starts with "/". This should never
			// happen but we try to recover from it by trimming leading
			// slashes and checking whether we still have a key.
			kvp.Key = strings.TrimLeft(kvp.Key, "/")
			if len(strings.TrimSpace(kvp.Key)) == 0 {
				continue //Nothing to do here with an empty key
			}
			i = strings.IndexByte(kvp.Key, '/')
		}
		if i == -1 {
			// This key has no separator, so it has no prefix, so it's
			// a leaf in our tree.
			// We convert its value into a configuration.String and
			// we're done.
			m[kvp.Key] = String(kvp.Value)
		} else {
			// A separator was found. If the Consul output is in any way
			// legit, i cannot be 0
			prefix := kvp.Key[:i]
			kvp.Key = kvp.Key[i+1:]
			if prefixSet[prefix] == nil {
				prefixSet[prefix] = make(api.KVPairs, 0)
			}
			prefixSet[prefix] = append(prefixSet[prefix], kvp)
		}
	}
	for prefix, kvpairslist := range prefixSet {
		m[prefix] = mapify(kvpairslist)
	}
	return m
}
