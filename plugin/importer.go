// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/drone/drone-go/drone"

	"github.com/google/go-github/github"
	"github.com/google/go-jsonnet"
)

type importer struct {
	client *github.Client
	repo   drone.Repo
	build  drone.Build

	// jsonnet does not cache file imports and may request
	// the same file multiple times. We cache the files to
	// duplicate API calls.
	cache map[string]string

	// limit the number of outbound requests. github limits
	// the number of api requests per hour, so we should
	// make sure that a single build does not abuse the api
	// by imporing dozens of files.
	limit int

	// counts the number of outbound requests. if the count
	// exceeds the limit, the importer will return errors.
	count int
}

func (i *importer) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	if i.cache == nil {
		i.cache = map[string]string{}
	}

	// the import is relative to the imported from path. the
	// imported path must resolve to a filepath relative to
	// the root of the repository.
	importedPath = path.Join(
		path.Dir(importedFrom),
		importedPath,
	)
	if strings.HasPrefix(importedFrom, "../") {
		err = fmt.Errorf("jsonnet: cannot resolve import: %s", importedPath)
		return contents, foundAt, err
	}

	// if the contents exist in the cache, return the
	// cached item.
	if contents, ok := i.cache[importedPath]; ok {
		return jsonnet.MakeContents(contents), importedPath, nil
	}

	defer func() {
		i.count++
	}()

	// if the import limit is exceeded log an error message.
	if i.limit > 0 && i.count > i.limit {
		return contents, foundAt, errors.New("jsonnet: import limit exceeded")
	}

	// get the configuration file from the github repository
	// for the build ref.
	opts := &github.RepositoryContentGetOptions{Ref: i.build.After}
	data, _, _, err := i.client.Repositories.GetContents(context.Background(), i.repo.Namespace, i.repo.Name, importedPath, opts)
	if err != nil {
		return contents, foundAt, err
	}

	// extracts and decodes the base64-encoded file contents
	// from the response body.
	content, err := data.GetContent()
	if err != nil {
		return contents, foundAt, err
	}

	// caches the contents for future API calls.
	i.cache[importedPath] = content

	return jsonnet.MakeContents(content), importedPath, err
}
