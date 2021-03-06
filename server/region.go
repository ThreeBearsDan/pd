// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bytes"
	"log"

	"github.com/google/btree"
	"github.com/pingcap/kvproto/pkg/metapb"
)

var _ btree.Item = &regionItem{}

type regionItem struct {
	region *metapb.Region
}

// Less returns true if the region start key is greater than the other.
// So we will sort the region with start key reversely.
func (r *regionItem) Less(other btree.Item) bool {
	left := r.region.GetStartKey()
	right := other.(*regionItem).region.GetStartKey()
	return bytes.Compare(left, right) > 0
}

func (r *regionItem) Contains(key []byte) bool {
	start, end := r.region.GetStartKey(), r.region.GetEndKey()
	return bytes.Compare(key, start) >= 0 && (len(end) == 0 || bytes.Compare(key, end) < 0)
}

const (
	defaultBTreeDegree = 64
)

type regionTree struct {
	tree *btree.BTree
}

func newRegionTree() *regionTree {
	return &regionTree{
		tree: btree.New(defaultBTreeDegree),
	}
}

func (t *regionTree) length() int {
	return t.tree.Len()
}

func (t *regionTree) insert(region *metapb.Region) {
	item := &regionItem{
		region: region,
	}

	oldItem := t.tree.ReplaceOrInsert(item)
	if oldItem != nil {
		log.Panicf("insert existed region: %v", region)
	}
}

func (t *regionTree) update(region *metapb.Region) {
	item := &regionItem{
		region: region,
	}

	oldItem := t.tree.ReplaceOrInsert(item)
	if oldItem == nil {
		log.Panicf("update non exist region: %v", region)
	}
}

func (t *regionTree) remove(region *metapb.Region) {
	item := &regionItem{
		region: region,
	}

	oldItem := t.tree.Delete(item)
	if oldItem == nil {
		log.Panicf("remove non exist region: %v", region)
	}
}

func (t *regionTree) search(regionKey []byte) *metapb.Region {
	item := &regionItem{
		region: &metapb.Region{StartKey: regionKey},
	}

	var result *regionItem
	t.tree.AscendGreaterOrEqual(item, func(i btree.Item) bool {
		result = i.(*regionItem)
		return false
	})

	if result == nil || !result.Contains(regionKey) {
		return nil
	}

	return result.region
}
