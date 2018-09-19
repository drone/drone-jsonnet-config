// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/config"

	"github.com/google/go-github/github"
	"github.com/google/go-jsonnet"
	"golang.org/x/oauth2"
)

// New returns a new jsonnet configuration plugin.
func New(server, token string) config.Plugin {
	if server != "" {
		server = strings.TrimPrefix(server, "/")
		server = server + "/"
	}
	return &plugin{
		server: server,
		token:  token,
	}
}

type plugin struct {
	server string
	token  string
}

func (p *plugin) Find(ctx context.Context, req *config.Request) (*drone.Config, error) {
	var client *github.Client

	// creates a github transport that authenticates
	// http requests using the github access token.
	trans := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.token},
	))

	// if a custom github endpoint is configured, for use
	// with github enterprise, we need to adjust the client
	// url accordingly.
	if p.server == "" {
		client = github.NewClient(trans)
	} else {
		url, err := url.Parse(p.server)
		if err != nil {
			return nil, err
		}
		// TODO(bradrydzewski) upgrade go-github and use
		// github.NewEnterpriseClient
		client = github.NewClient(trans)
		client.BaseURL = url
	}

	// HACK: the drone-go library does not currently work
	// with 0.9 which means the configuration file path is
	// always empty. default to .drone.yml. This can be
	// removed as soon as drone-go is fully updated for 0.9.
	path := req.Repo.Config
	if path == "" {
		path = ".drone.jsonnet"
	}

	// get the configuration file from the github repository
	// for the build ref.
	opts := &github.RepositoryContentGetOptions{Ref: req.Build.After}
	data, _, _, err := client.Repositories.GetContents(req.Repo.Namespace, req.Repo.Name, path, opts)
	if err != nil {
		return nil, err
	}

	// if there is no error and no content, a nil value is
	// returned. The plugin responds with a 204 No Content,
	// instrucing Drone to fallback to a .drone.yml file.
	if data == nil {
		return nil, nil
	}

	// get the file contents.
	content, err := data.GetContent()
	if err != nil {
		return nil, err
	}

	// create the jsonnet vm
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500
	vm.StringOutput = false
	vm.ErrorFormatter.SetMaxStackTraceSize(20)
	vm.Importer(
		&importer{
			client: client,
			repo:   req.Repo,
			build:  req.Build,
			limit:  10,
		},
	)

	// convert the jsonnet file to yaml
	buf := new(bytes.Buffer)
	docs, err := vm.EvaluateSnippetStream(path, content)
	if err != nil {
		return nil, err
	}

	// the jsonnet vm returns a stream of yaml documents
	// that need to be combined into a single yaml file.
	for _, doc := range docs {
		buf.WriteString("---")
		buf.WriteString("\n")
		buf.WriteString(doc)
	}

	return &drone.Config{
		Data: buf.String(),
	}, nil
}
