/*
 * Copyright 2020 Sheaf Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sheaf

import (
	"github.com/dprotaso/go-yit"
	"github.com/pivotal/image-relocation/pkg/image"
	"gopkg.in/yaml.v3"
)

func replaceImage(manifest []byte, imageMap map[image.Name]image.Name) []byte {
	var doc yaml.Node
	err := yaml.Unmarshal(manifest, &doc)
	if err != nil {
		panic(err) // FIXME: return error
	}

	it := yit.FromNode(&doc).
		RecurseNodes().
		Filter(podSpecPredicate)

	// We're now iterating over nodes matching the above pod shape
	for podspec, ok := it(); ok; podspec, ok = it() {

		imageit := yit.FromNode(podspec).
			ValuesForMap(
				// key predicate
				yit.WithStringValue("spec"),
				// value predicate
				specPredicate,
			).
			ValuesForMap(
				// key predicate
				yit.WithStringValue("containers"),
				// value predicate
				containerPredicate,
			).
			Values(). // iterate over all container elements
			ValuesForMap(yit.WithStringValue("image"), yit.StringValue)

		for image, iok := imageit(); iok; image, iok = imageit() {
			for oldImage, newImage := range imageMap {
				for _, oi := range oldImage.Synonyms() {
					if image.Value == oi.String() {
						image.Value = newImage.String()
					}
				}
			}
		}

	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		panic(err) // FIXME: return error
	}

	return out
}

var data = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-deployment
spec:
  template:
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
      - image: nginy
        name: nginy
        ports:
        - containerPort: 81
`

var containerPredicate = yit.Intersect(
	yit.WithKind(yaml.SequenceNode),

	WithNestedValue(yit.WithMapKeyValue(
		// key predicate
		yit.WithStringValue("image"),

		// value predicate
		yit.StringValue,
	)),
)

var specPredicate = yit.Intersect(
	yit.WithKind(yaml.MappingNode),

	yit.WithMapKeyValue(
		// key predicate
		yit.WithValue("containers"),

		// value predicate
		containerPredicate,
	),
)

var podSpecPredicate = yit.Intersect(
	yit.WithKind(yaml.MappingNode),
	yit.WithMapKeyValue(
		// key predicate
		yit.WithValue("spec"),

		// value predicate
		specPredicate,
	),
)

func WithNestedValue(p yit.Predicate) yit.Predicate {
	return func(node *yaml.Node) bool {
		return yit.FromNode(node).Values().AnyMatch(p)
	}
}
