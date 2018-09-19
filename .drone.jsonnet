local pipeline(name, os, arch) = {
    kind: "pipeline",
    name: name,
    platform: {
        os: os,
        arch: arch,
    },
    steps: [
        {
            name: "build",
            image: "golang",
            environment: {
                "GOOS": os,
                "GOARCH": arch,
                "CGO_ENABLED": "0",
            },
            commands: [
                "go get -u github.com/golang/dep/cmd/dep",
                "dep ensure",
                "go test ./...",
                "go build -o release/"+os+"/"+arch+"/drone-jsonnet-config github.com/drone/drone-jsonnet-config/cmd/drone-jsonnet-config",
            ],
        },
        {
            name: "publish",
            image: "plugins/docker",
            settings: {
                repo: "drone/drone-jsonnet",
                auto_tag: true,
                auto_tag_suffix: os + "-" + arch,
                username: { "$secret": "username" },
                password: { "$secret": "password" },
                dockerfile: "docker/Dockerfile." + os + "." + arch,
            }, 
        },
    ],
};

local manifest = {
    kind: "pipeline",
    name: "manifest",
    steps: [
        {
            name: "upload",
            image: "plugins/manifest",
            settings: {
                spec: "manifest.tmpl",
                auto_tag: true,
                ignore_missing: true,
            },
        },
    ],
    depends_on: [
        "amd64",
        "arm64",
        "arm"
    ],
};

local secrets = {
    kind: "secret",
    type: "external",
    data: {
        "username": "drone/docker#username",
        "password": "drone/docker#password"
    },
};

[
  pipeline("amd64", "linux", "amd64"),
  #pipeline("arm64", "linux", "arm64"),
  #pipeline("arm", "linux", "arm"),
  #manifest,
  secrets,
]