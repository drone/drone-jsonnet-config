// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone-go/drone"

	"github.com/google/go-github/github"
	"github.com/google/go-jsonnet"
)

func TestImport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/contents.json")
		w.Write(out)
	}))
	defer ts.Close()

	client, err := github.NewEnterpriseClient(ts.URL, ts.URL, nil)
	if err != nil {
		t.Error(err)
		return
	}

	vm := jsonnet.MakeVM()
	vm.Importer(
		&importer{
			client: client,
			repo: drone.Repo{
				Slug: "octocat/hello-world",
			},
			build: drone.Build{
				After: "3d21ec53a331a6f037a91c368710b99387d012c1",
			},
		},
	)

	out, err := ioutil.ReadFile("testdata/imports.jsonnet")
	if err != nil {
		t.Error(err)
		return
	}

	_, err = vm.EvaluateSnippetStream("testdata/contents.jsonnet", string(out))
	if err != nil {
		t.Error(err)
		return
	}
}
