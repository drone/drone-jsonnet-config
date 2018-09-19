// Copyright 2018 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/config"
)

// empty context
var noContext = context.Background()

// mock github token
const mockToken = "d7c559e677ebc489d4e0193c8b97a12e"

func TestPlugin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, _ := ioutil.ReadFile("testdata/contents.json")
		w.Write(out)
	}))
	defer ts.Close()

	req := &config.Request{
		Build: drone.Build{
			After: "3d21ec53a331a6f037a91c368710b99387d012c1",
		},
		Repo: drone.Repo{
			Slug: "octocat/hello-world",
		},
	}

	plugin := New(ts.URL, mockToken)
	config, err := plugin.Find(noContext, req)
	if err != nil {
		t.Error(err)
		return
	}

	data, err := ioutil.ReadFile("testdata/contents.jsonnet.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	if want, got := string(data), config.Data; want != got {
		t.Errorf("Want %q got %q", want, got)
	}
}
