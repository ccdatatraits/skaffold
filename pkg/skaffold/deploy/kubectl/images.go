/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubectl

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

// for testing
var warner Warner = &logrusWarner{}

// ReplaceImages replaces image names in a list of manifests.
func (l *ManifestList) ReplaceImages(builds []build.Artifact) (ManifestList, error) {
	replacer := newImageReplacer(builds)

	updated, err := l.Visit(replacer)
	if err != nil {
		return nil, errors.Wrap(err, "replacing images")
	}

	replacer.Check()
	logrus.Debugln("manifests with tagged images", updated.String())

	return updated, nil
}

type imageReplacer struct {
	tagsByImageName map[string]string
	found           map[string]bool
}

func newImageReplacer(builds []build.Artifact) *imageReplacer {
	tagsByImageName := make(map[string]string)
	for _, build := range builds {
		tagsByImageName[build.ImageName] = build.Tag
	}

	return &imageReplacer{
		tagsByImageName: tagsByImageName,
		found:           make(map[string]bool),
	}
}

func (r *imageReplacer) Matches(key string) bool {
	return key == "image"
}

func (r *imageReplacer) NewValue(key string, old interface{}) (bool, interface{}) {
	image := old.(string)

	parsed, err := docker.ParseReference(image)
	if err != nil {
		warner.Warnf("Couldn't parse image: %s", image)
		return false, nil
	}

	if tag, present := r.tagsByImageName[parsed.BaseName]; present {
		if parsed.FullyQualified {
			if tag == image {
				r.found[parsed.BaseName] = true
			}
		} else {
			r.found[parsed.BaseName] = true
			return true, tag
		}
	}

	return false, nil
}

func (r *imageReplacer) Check() {
	for imageName := range r.tagsByImageName {
		if !r.found[imageName] {
			warner.Warnf("image [%s] is not used by the deployment", imageName)
		}
	}
}