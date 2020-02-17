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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/bryanl/sheaf/pkg/images"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/util/jsonpath"
)

// ContainerImages returns images from containers in manifest path
func ContainerImages(manifestPath string) (images.Set, error) {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return images.Empty, fmt.Errorf("read file: %w", err)
	}

	r := bytes.NewReader(data)
	decoder := yaml.NewYAMLOrJSONDecoder(r, 4096)

	imgs := images.Empty

	for {
		var m map[string]interface{}
		if err := decoder.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return images.Empty, fmt.Errorf("decode failed: %w", err)
		}

		j := jsonpath.New("parser")
		if err := j.Parse("{range ..spec.containers[*]}{.image}{','}{end}"); err != nil {
			return images.Empty, fmt.Errorf("unable to parse: %w", err)
		}

		var buf bytes.Buffer
		if err := j.Execute(&buf, m); err != nil {
			// jsonpath doesn't return a helpful error, so looking at the error message
			if strings.Contains(err.Error(), "is not found") {
				continue
			}
			return images.Empty, fmt.Errorf("search manifest for containers: %w", err)
		}

		bufImages, err := images.New(filterEmpty(strings.Split(buf.String(), ",")))
		if err != nil {
			return images.Empty, err
		}
		imgs = imgs.Union(bufImages)
	}

	return imgs, nil
}

func filterEmpty(ss []string) []string {
	result := []string{}
	for _, s := range ss {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
