FROM docker.io/golang:1.9.2

copy src src

RUN cd {{BASE_PATH}}; go install -ldflags '-X "{{PACKAGE_IN_VENDOR}}/debug.ServiceKey={{KEY}}" -X "{{PACKAGE_IN_VENDOR}}/debug.GitHash={{GIT_HASH}}" -X "{{PACKAGE_IN_VENDOR}}/debug.GitTime={{GIT_TIME}}" -X "{{PACKAGE_IN_VENDOR}}/debug.GitMessage={{GIT_MESSAGE}}" -X "{{PACKAGE_IN_VENDOR}}/debug.Project={{PROJECT}}"'
